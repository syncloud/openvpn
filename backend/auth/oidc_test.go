package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newOIDC() *OIDC {
	return &OIDC{CookieSecret: []byte("test-secret")}
}

func TestSession_RoundTrip(t *testing.T) {
	o := newOIDC()

	cookie, err := o.encodeSession(session{Sub: "user", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	assert.True(t, o.ValidSession(cookie))
}

func TestSession_Expired(t *testing.T) {
	o := newOIDC()

	cookie, err := o.encodeSession(session{Sub: "user", Exp: time.Now().Add(-time.Hour).Unix()})
	require.NoError(t, err)

	assert.False(t, o.ValidSession(cookie))
}

func TestSession_TamperedBody(t *testing.T) {
	o := newOIDC()
	cookie, err := o.encodeSession(session{Sub: "user", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	forged, err := (&OIDC{CookieSecret: []byte("other-secret")}).encodeSession(
		session{Sub: "attacker", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	assert.False(t, o.ValidSession(forged))
	assert.NotEqual(t, cookie, forged)
}

func TestSession_Malformed(t *testing.T) {
	o := newOIDC()

	for _, raw := range []string{"", "nodot", "!!!.???", "."} {
		assert.False(t, o.ValidSession(raw), raw)
	}
}

func TestMiddleware_ApiUnauthenticatedIs401(t *testing.T) {
	o := newOIDC()
	handler := o.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/clients", nil))

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestMiddleware_PageUnauthenticatedRedirects(t *testing.T) {
	o := newOIDC()
	handler := o.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "/auth/login", recorder.Header().Get("Location"))
}

func TestMiddleware_ValidSessionPassesThrough(t *testing.T) {
	o := newOIDC()
	cookie, err := o.encodeSession(session{Sub: "user", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	called := false
	handler := o.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/clients", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	handler.ServeHTTP(httptest.NewRecorder(), request)

	assert.True(t, called)
}

func TestPkceChallenge_IsDeterministicAndUrlSafe(t *testing.T) {
	challenge := pkceChallenge("verifier")

	assert.Equal(t, challenge, pkceChallenge("verifier"))
	assert.NotContains(t, challenge, "+")
	assert.NotContains(t, challenge, "/")
	assert.NotContains(t, challenge, "=")
}

func TestPlatformCAClient_TrustsPlatformCA(t *testing.T) {
	dir := t.TempDir()
	caPath := path.Join(dir, "syncloud.ca.crt")

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(7),
		Subject:               pkix.Name{CommonName: "syncloud"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(caPath,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644))

	client, err := platformCAClient(caPath)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestPlatformCAClient_MissingFileIsAnError(t *testing.T) {
	_, err := platformCAClient(path.Join(t.TempDir(), "absent.crt"))
	assert.Error(t, err, "a missing platform CA must be loud, not a silent fallback to system roots")
}

func TestPlatformCAClient_GarbageIsAnError(t *testing.T) {
	caPath := path.Join(t.TempDir(), "bad.crt")
	require.NoError(t, os.WriteFile(caPath, []byte("not a certificate"), 0644))

	_, err := platformCAClient(caPath)
	assert.Error(t, err)
}

func TestPlatformCAClient_EmptyPathUsesSystemRoots(t *testing.T) {
	client, err := platformCAClient("")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestInitWithRetry_DoesNotBlockServingWhenAuthUnreachable(t *testing.T) {
	o := &OIDC{
		IssuerURL:    "https://auth.unresolvable.invalid",
		CookieSecret: []byte("s"),
		CAPath:       "",
		AuthSocket:   path.Join(t.TempDir(), "absent.socket"),
		Logger:       zap.NewNop(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o.InitWithRetry(ctx, time.Hour)

	assert.False(t, o.Ready(), "discovery cannot succeed against an unresolvable host")

	recorder := httptest.NewRecorder()
	o.Login(recorder, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code,
		"login must report unavailable, not panic on a nil oauth2 config")

	cb := httptest.NewRecorder()
	o.Callback(cb, httptest.NewRequest(http.MethodGet, "/auth/callback", nil))
	assert.Equal(t, http.StatusServiceUnavailable, cb.Code)
}

func TestSessionsKeepWorkingWhileOidcIsDown(t *testing.T) {
	o := &OIDC{CookieSecret: []byte("secret"), Logger: zap.NewNop()}
	cookie, err := o.encodeSession(session{Sub: "u", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	called := false
	handler := o.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "/api/clients", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	assert.False(t, o.Ready())
	assert.True(t, called, "an existing session is validated from the cookie HMAC, not from Authelia")
}

func TestAuthTransport_OverSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "a")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	socket := path.Join(dir, "s")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	defer listener.Close()

	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"served":"over-socket","host":"` + r.Host + `"}`))
	}))

	rt := newAuthTransport("https://auth.example.com", socket, http.DefaultTransport)
	client := &http.Client{Transport: rt, Timeout: 10 * time.Second}

	resp, err := client.Get("https://auth.example.com/.well-known/openid-configuration")
	require.NoError(t, err, "must not need DNS for the auth host")
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "over-socket")
	assert.Contains(t, string(body), "auth.example.com",
		"the public host must be preserved so token issuer validation still matches")
}

func TestAuthTransport_NoSocketFallsBackToNetwork(t *testing.T) {
	rt := newAuthTransport("https://auth.example.com", "", http.DefaultTransport)
	assert.Equal(t, http.DefaultTransport, rt, "with no socket configured it must not wrap")
}
