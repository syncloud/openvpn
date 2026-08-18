package auth

import (
	"net/http"
	"net/http/httptest"
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
