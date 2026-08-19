package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"backend/mi"
	"backend/pki"
	"backend/server"
)

type Api struct {
	PKI     *pki.PKI
	Server  *server.Service
	MI      *mi.Client
	DataDir string
	Logger  *zap.Logger
}

type createRequest struct {
	Name string `json:"name"`
}

type statusResponse struct {
	Connections   []mi.Connection `json:"connections"`
	Port          int             `json:"port"`
	Proto         string          `json:"proto"`
	ServerAddress string          `json:"server_address"`
	Version       string          `json:"version"`
	Running       bool            `json:"running"`
}

func (a *Api) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/clients", a.listClients)
	mux.HandleFunc("POST /api/clients", a.createClient)
	mux.HandleFunc("DELETE /api/clients/{name}", a.revokeClient)
	mux.HandleFunc("GET /api/clients/{name}/config", a.clientConfig)
	mux.HandleFunc("GET /api/status", a.status)
	mux.HandleFunc("GET /api/settings", a.getSettings)
	mux.HandleFunc("POST /api/settings", a.updateSettings)
}

func (a *Api) listClients(w http.ResponseWriter, _ *http.Request) {
	clients, err := a.PKI.List()
	if err != nil {
		a.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, clients)
}

func (a *Api) createClient(w http.ResponseWriter, r *http.Request) {
	var request createRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		a.fail(w, http.StatusBadRequest, err)
		return
	}
	if !pki.ValidName(request.Name) {
		a.fail(w, http.StatusBadRequest, fmt.Errorf("invalid name"))
		return
	}

	switch err := a.PKI.IssueClient(request.Name); {
	case errors.Is(err, pki.ErrExists):
		a.fail(w, http.StatusConflict, err)
	case err != nil:
		a.fail(w, http.StatusInternalServerError, err)
	default:
		writeJSON(w, http.StatusCreated, map[string]string{"name": request.Name})
	}
}

func (a *Api) revokeClient(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	switch err := a.PKI.Revoke(name); {
	case errors.Is(err, pki.ErrNotFound):
		a.fail(w, http.StatusNotFound, err)
	case err != nil:
		a.fail(w, http.StatusInternalServerError, err)
	default:
		a.restartAfterRevoke()
		writeJSON(w, http.StatusOK, map[string]string{"name": name})
	}
}

func (a *Api) restartAfterRevoke() {
	settings, err := server.LoadSettings(a.DataDir)
	if err != nil {
		a.Logger.Error("load settings after revoke", zap.Error(err))
		return
	}
	if err := a.Server.Apply(settings); err != nil {
		a.Logger.Error("apply after revoke", zap.Error(err))
	}
}

func (a *Api) clientConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	settings, err := server.LoadSettings(a.DataDir)
	if err != nil {
		a.fail(w, http.StatusInternalServerError, err)
		return
	}
	config, err := a.Server.ClientConfig(name, settings)
	if err != nil {
		a.fail(w, http.StatusNotFound, err)
		return
	}

	w.Header().Set("Content-Type", "application/x-openvpn-profile")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name+".ovpn"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(config))
}

func (a *Api) status(w http.ResponseWriter, _ *http.Request) {
	settings, err := server.LoadSettings(a.DataDir)
	if err != nil {
		a.fail(w, http.StatusInternalServerError, err)
		return
	}

	response := statusResponse{
		Connections:   []mi.Connection{},
		Port:          settings.Port,
		Proto:         settings.Proto,
		ServerAddress: a.Server.ServerAddress,
	}
	if connections, err := a.MI.Connections(); err != nil {
		a.Logger.Warn("openvpn management interface unavailable", zap.Error(err))
	} else {
		response.Connections = connections
		response.Running = true
	}
	if version, err := a.MI.Version(); err == nil {
		response.Version = version
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *Api) getSettings(w http.ResponseWriter, _ *http.Request) {
	settings, err := server.LoadSettings(a.DataDir)
	if err != nil {
		a.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (a *Api) updateSettings(w http.ResponseWriter, r *http.Request) {
	current, err := server.LoadSettings(a.DataDir)
	if err != nil {
		a.fail(w, http.StatusInternalServerError, err)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&current); err != nil {
		a.fail(w, http.StatusBadRequest, err)
		return
	}
	if err := current.Validate(); err != nil {
		a.fail(w, http.StatusBadRequest, err)
		return
	}
	if err := a.Server.Apply(current); err != nil {
		a.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, current)
}

func (a *Api) fail(w http.ResponseWriter, code int, err error) {
	if code >= http.StatusInternalServerError {
		a.Logger.Error("request failed", zap.Int("code", code), zap.Error(err))
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
