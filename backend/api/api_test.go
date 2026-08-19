package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"backend/mi"
	"backend/pki"
	"backend/server"
)

func newApi(t *testing.T) (*Api, *http.ServeMux) {
	t.Helper()
	dataDir := t.TempDir()
	certs := &pki.PKI{Dir: path.Join(dataDir, "pki")}
	require.NoError(t, certs.Init())

	service := &server.Service{
		DataDir:          dataDir,
		OpenvpnDir:       path.Join(dataDir, "openvpn"),
		TemplatesDir:     path.Join("..", "..", "templates"),
		ManagementSocket: path.Join(dataDir, "openvpn.socket"),
		ServerAddress:    "device.syncloud.it",
		PKI:              certs,
	}

	api := &Api{
		PKI:     certs,
		Server:  service,
		MI:      &mi.Client{Socket: path.Join(dataDir, "absent.socket")},
		DataDir: dataDir,
		Logger:  zap.NewNop(),
	}
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	return api, mux
}

func do(t *testing.T, mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(method, target, reader))
	return recorder
}

func TestListClients_Empty(t *testing.T) {
	_, mux := newApi(t)

	response := do(t, mux, http.MethodGet, "/api/clients", "")

	assert.Equal(t, http.StatusOK, response.Code)
	var clients []pki.Client
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &clients))
	assert.Empty(t, clients)
}

func TestCreateAndListClient(t *testing.T) {
	_, mux := newApi(t)

	created := do(t, mux, http.MethodPost, "/api/clients", `{"name":"laptop"}`)
	assert.Equal(t, http.StatusCreated, created.Code)

	listed := do(t, mux, http.MethodGet, "/api/clients", "")
	var clients []pki.Client
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &clients))
	require.Len(t, clients, 1)
	assert.Equal(t, "laptop", clients[0].Name)
}

func TestCreateClient_Duplicate(t *testing.T) {
	_, mux := newApi(t)
	do(t, mux, http.MethodPost, "/api/clients", `{"name":"laptop"}`)

	response := do(t, mux, http.MethodPost, "/api/clients", `{"name":"laptop"}`)

	assert.Equal(t, http.StatusConflict, response.Code)
}

func TestCreateClient_InvalidName(t *testing.T) {
	_, mux := newApi(t)

	for _, name := range []string{"", "../escape", "with space", "server"} {
		body, err := json.Marshal(createRequest{Name: name})
		require.NoError(t, err)
		response := do(t, mux, http.MethodPost, "/api/clients", string(body))
		assert.Equal(t, http.StatusBadRequest, response.Code, name)
	}
}

func TestClientConfig_Downloads(t *testing.T) {
	_, mux := newApi(t)
	do(t, mux, http.MethodPost, "/api/clients", `{"name":"laptop"}`)

	response := do(t, mux, http.MethodGet, "/api/clients/laptop/config", "")

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Header().Get("Content-Disposition"), `"laptop.ovpn"`)
	assert.Contains(t, response.Body.String(), "remote device.syncloud.it 1194")
	assert.Contains(t, response.Body.String(), "BEGIN PRIVATE KEY")
}

func TestClientConfig_Unknown(t *testing.T) {
	_, mux := newApi(t)

	response := do(t, mux, http.MethodGet, "/api/clients/nobody/config", "")

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestRevokeClient(t *testing.T) {
	_, mux := newApi(t)
	do(t, mux, http.MethodPost, "/api/clients", `{"name":"lost"}`)

	response := do(t, mux, http.MethodDelete, "/api/clients/lost", "")
	assert.Equal(t, http.StatusOK, response.Code)

	listed := do(t, mux, http.MethodGet, "/api/clients", "")
	var clients []pki.Client
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &clients))
	require.Len(t, clients, 1)
	assert.True(t, clients[0].Revoked)

	config := do(t, mux, http.MethodGet, "/api/clients/lost/config", "")
	assert.Equal(t, http.StatusNotFound, config.Code)
}

func TestRevokeClient_Unknown(t *testing.T) {
	_, mux := newApi(t)

	response := do(t, mux, http.MethodDelete, "/api/clients/nobody", "")

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestSettings_GetDefaults(t *testing.T) {
	_, mux := newApi(t)

	response := do(t, mux, http.MethodGet, "/api/settings", "")

	assert.Equal(t, http.StatusOK, response.Code)
	var settings server.Settings
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
	assert.Equal(t, server.DefaultPort, settings.Port)
}

func TestSettings_UpdatePort(t *testing.T) {
	_, mux := newApi(t)

	response := do(t, mux, http.MethodPost, "/api/settings", `{"port":1195}`)
	assert.Equal(t, http.StatusOK, response.Code)

	updated := do(t, mux, http.MethodGet, "/api/settings", "")
	var settings server.Settings
	require.NoError(t, json.Unmarshal(updated.Body.Bytes(), &settings))
	assert.Equal(t, 1195, settings.Port)
	assert.Equal(t, server.DefaultDataCiphers, settings.DataCiphers)
}

func TestSettings_RejectsInvalidPort(t *testing.T) {
	_, mux := newApi(t)

	response := do(t, mux, http.MethodPost, "/api/settings", `{"port":70000}`)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestStatus_ServerDown(t *testing.T) {
	_, mux := newApi(t)

	response := do(t, mux, http.MethodGet, "/api/status", "")

	assert.Equal(t, http.StatusOK, response.Code)
	var status statusResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &status))
	assert.False(t, status.Running)
	assert.Empty(t, status.Connections)
	assert.Equal(t, server.DefaultPort, status.Port)
	assert.Equal(t, "device.syncloud.it", status.ServerAddress)
}
