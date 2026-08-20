package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
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
	CAPath       string
	AuthSocket   string
	Logger       *zap.Logger

	client       *http.Client
	mu           sync.RWMutex
	ready        bool
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
}

func (o *OIDC) Init(ctx context.Context) error {
	network, err := platformCAClient(o.CAPath)
	if err != nil {
		return err
	}

	candidates := []*http.Client{}
	if o.AuthSocket != "" {
		candidates = append(candidates, &http.Client{
			Timeout:   30 * time.Second,
			Transport: newAuthTransport(o.IssuerURL, o.AuthSocket, network.Transport),
		})
	}
	candidates = append(candidates, network)

	var provider *oidc.Provider
	var lastErr error
	for _, client := range candidates {
		o.client = client
		provider, lastErr = oidc.NewProvider(o.context(ctx), o.IssuerURL)
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return fmt.Errorf("oidc provider discovery: %w", lastErr)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.provider = provider
	o.verifier = provider.Verifier(&oidc.Config{ClientID: o.ClientID})
	o.oauth2Config = oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		RedirectURL:  o.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}
	o.ready = true
	return nil
}

// Discovery must never be fatal. It reaches Authelia over the platform's local
// socket, but on a device whose resolver cannot answer for its own domain the
// public fallback fails, and a hard exit here crash-loops the backend and
// leaves nginx serving 502 for the whole app. Existing sessions keep working
// while this retries, because they are validated from the cookie HMAC alone.
func (o *OIDC) InitWithRetry(ctx context.Context, every time.Duration) {
	if err := o.Init(ctx); err == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(every):
			}
			if err := o.Init(ctx); err != nil {
				o.Logger.Warn("oidc discovery failed, retrying", zap.Error(err))
				continue
			}
			o.Logger.Info("oidc discovery succeeded")
			return
		}
	}()
}

func (o *OIDC) Ready() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.ready
}

// The public auth hostname may be unresolvable from the device itself, so
// server-to-Authelia calls are dialled over the platform's unix socket while
// keeping the public issuer in the URL, which is what token validation checks.
type authTransport struct {
	host     string
	socket   string
	overSock http.RoundTripper
	fallback http.RoundTripper
}

func newAuthTransport(issuerURL, socket string, fallback http.RoundTripper) http.RoundTripper {
	host := ""
	if u, err := url.Parse(issuerURL); err == nil {
		host = u.Hostname()
	}
	if host == "" || socket == "" {
		return fallback
	}
	return &authTransport{
		host:   host,
		socket: socket,
		overSock: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socket)
			},
		},
		fallback: fallback,
	}
}

// Authelia derives the OIDC issuer from the forwarded headers the public vhost
// normally adds. Without them it advertises http://localhost, which fails
// discovery validation against the public issuer, so they are set explicitly.
func (t *authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Hostname() != t.host {
		return t.fallback.RoundTrip(r)
	}
	viaSocket := r.Clone(r.Context())
	viaSocket.URL.Scheme = "http"
	viaSocket.Host = t.host
	viaSocket.Header.Set("X-Forwarded-Proto", "https")
	viaSocket.Header.Set("X-Forwarded-Host", t.host)
	return t.overSock.RoundTrip(viaSocket)
}

func (o *OIDC) Login(w http.ResponseWriter, r *http.Request) {
	if !o.Ready() {
		http.Error(w, "sign-in is unavailable: the device cannot reach its auth service yet",
			http.StatusServiceUnavailable)
		return
	}
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
	if !o.Ready() {
		http.Error(w, "sign-in is unavailable: the device cannot reach its auth service yet",
			http.StatusServiceUnavailable)
		return
	}
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

	ctx := o.context(r.Context())
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

// The platform terminates TLS with its own CA, which is not installed into the
// OS trust store, so Go's default verification rejects Authelia. Every call to
// the provider goes through a client that trusts it in addition to system roots.
func platformCAClient(caPath string) (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if caPath != "" {
		pem, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("read platform ca %s: %w", caPath, err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("platform ca %s has no usable certificate", caPath)
		}
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}},
	}, nil
}

func (o *OIDC) context(ctx context.Context) context.Context {
	if o.client == nil {
		return ctx
	}
	return oidc.ClientContext(ctx, o.client)
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
