package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

const (
	stateCookie    = "openvpn_oidc_state"
	verifierCookie = "openvpn_oidc_verifier"
	sessionCookie  = "openvpn_session"
	sessionTTL     = 12 * time.Hour
	flowTTL        = 10 * time.Minute
)

type OIDC struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AdminGroup   string
	CookieSecret []byte
	Logger       *zap.Logger

	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
}

func (o *OIDC) Init(ctx context.Context) error {
	provider, err := oidc.NewProvider(ctx, o.IssuerURL)
	if err != nil {
		return fmt.Errorf("oidc provider discovery: %w", err)
	}
	o.provider = provider
	o.verifier = provider.Verifier(&oidc.Config{ClientID: o.ClientID})
	o.oauth2Config = oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		RedirectURL:  o.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}
	return nil
}

func (o *OIDC) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randBase64(16)
	if err != nil {
		http.Error(w, "state", http.StatusInternalServerError)
		return
	}
	verifier, err := randBase64(32)
	if err != nil {
		http.Error(w, "verifier", http.StatusInternalServerError)
		return
	}

	setCookie(w, stateCookie, state, flowTTL)
	setCookie(w, verifierCookie, verifier, flowTTL)

	http.Redirect(w, r, o.oauth2Config.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	), http.StatusFound)
}

func (o *OIDC) Callback(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie(stateCookie)
	if err != nil {
		http.Error(w, "state cookie missing", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	verifier, err := r.Cookie(verifierCookie)
	if err != nil {
		http.Error(w, "verifier cookie missing", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	token, err := o.oauth2Config.Exchange(ctx, r.URL.Query().Get("code"),
		oauth2.SetAuthURLParam("code_verifier", verifier.Value))
	if err != nil {
		o.Logger.Error("oauth2 exchange", zap.Error(err))
		http.Error(w, "exchange failed", http.StatusBadGateway)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "id_token missing", http.StatusBadGateway)
		return
	}
	idToken, err := o.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		o.Logger.Error("id token verify", zap.Error(err))
		http.Error(w, "id token invalid", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Sub    string   `json:"sub"`
		Email  string   `json:"email"`
		Groups []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "claims", http.StatusBadGateway)
		return
	}

	// Authelia returns groups in the userinfo response rather than the
	// id_token for some client registrations, so fall back to fetching it.
	if len(claims.Groups) == 0 {
		userInfo, err := o.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			o.Logger.Error("userinfo", zap.Error(err))
			http.Error(w, "userinfo failed", http.StatusBadGateway)
			return
		}
		var info struct {
			Groups []string `json:"groups"`
		}
		if err := userInfo.Claims(&info); err != nil {
			http.Error(w, "userinfo claims", http.StatusBadGateway)
			return
		}
		claims.Groups = info.Groups
	}

	if o.AdminGroup != "" && !slices.Contains(claims.Groups, o.AdminGroup) {
		o.Logger.Warn("access denied, user not in admin group",
			zap.String("sub", claims.Sub),
			zap.Strings("groups", claims.Groups),
			zap.String("required", o.AdminGroup))
		http.Error(w, "admin access only", http.StatusForbidden)
		return
	}

	cookie, err := o.encodeSession(session{
		Sub:   claims.Sub,
		Email: claims.Email,
		Exp:   time.Now().Add(sessionTTL).Unix(),
	})
	if err != nil {
		http.Error(w, "encode session", http.StatusInternalServerError)
		return
	}

	clearCookie(w, stateCookie)
	clearCookie(w, verifierCookie)
	setCookie(w, sessionCookie, cookie, sessionTTL)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (o *OIDC) Logout(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, sessionCookie)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (o *OIDC) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil || !o.ValidSession(cookie.Value) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "unauthenticated", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type session struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
}

func (o *OIDC) encodeSession(s session) (string, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(body) + "." +
		base64.URLEncoding.EncodeToString(o.sign(body)), nil
}

func (o *OIDC) ValidSession(raw string) bool {
	encodedBody, encodedSig, found := strings.Cut(raw, ".")
	if !found {
		return false
	}
	body, err := base64.URLEncoding.DecodeString(encodedBody)
	if err != nil {
		return false
	}
	sig, err := base64.URLEncoding.DecodeString(encodedSig)
	if err != nil {
		return false
	}
	if !hmac.Equal(sig, o.sign(body)) {
		return false
	}
	var s session
	if err := json.Unmarshal(body, &s); err != nil {
		return false
	}
	return time.Now().Unix() < s.Exp
}

func (o *OIDC) sign(body []byte) []byte {
	mac := hmac.New(sha256.New, o.CookieSecret)
	mac.Write(body)
	return mac.Sum(nil)
}

func setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	})
}

func randBase64(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
