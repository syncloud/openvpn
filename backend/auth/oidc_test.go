package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
