package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/syncloud/golib/platform"
	"go.uber.org/zap"

	"backend/api"
	"backend/auth"
	"backend/config"
	"backend/mi"
	"backend/pki"
	"backend/server"
)

const (
	app        = "openvpn"
	configFile = "/var/snap/" + app + "/current/config/backend.env"
	adminGroup = "syncloud"
)

func main() {
	cmd := &cobra.Command{
		Use:          "backend",
		Short:        "OpenVPN management API",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(logger())
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(log *zap.Logger) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load %s: %w", configFile, err)
	}

	certs := &pki.PKI{Dir: cfg.PkiDir}
	if err := certs.Init(); err != nil {
		return fmt.Errorf("init pki: %w", err)
	}

	settings, err := server.LoadSettings(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	service := &server.Service{
		DataDir:          cfg.DataDir,
		OpenvpnDir:       cfg.OpenvpnDir,
		TemplatesDir:     path.Join(cfg.AppDir, "templates"),
		ManagementSocket: cfg.ManagementSocket,
		ServerAddress:    cfg.ServerAddress,
		PKI:              certs,
		Restarter:        platform.New(),
	}
	if err := service.Render(settings); err != nil {
		return fmt.Errorf("render server config: %w", err)
	}
	log.Info("openvpn config rendered",
		zap.String("file", service.ServerConfPath()),
		zap.Int("port", settings.Port),
		zap.String("proto", settings.Proto))

	oidc := &auth.OIDC{
		IssuerURL:    cfg.OIDCAuthBaseURL,
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURI,
		AdminGroup:   adminGroup,
		CookieSecret: []byte(cfg.SessionSecret),
		Logger:       log,
	}
	if err := oidc.Init(context.Background()); err != nil {
		return fmt.Errorf("oidc init: %w", err)
	}

	handlers := &api.Api{
		PKI:     certs,
		Server:  service,
		MI:      &mi.Client{Socket: cfg.ManagementSocket},
		DataDir: cfg.DataDir,
		Logger:  log,
	}

	apiMux := http.NewServeMux()
	handlers.RegisterRoutes(apiMux)

	rootMux := http.NewServeMux()
	rootMux.HandleFunc("GET /auth/login", oidc.Login)
	rootMux.HandleFunc("GET /auth/callback", oidc.Callback)
	rootMux.HandleFunc("GET /auth/logout", oidc.Logout)
	rootMux.Handle("/api/", oidc.Middleware(apiMux))

	_ = os.Remove(cfg.Socket)
	listener, err := net.Listen("unix", cfg.Socket)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Socket, err)
	}
	if err := os.Chmod(cfg.Socket, 0666); err != nil {
		return fmt.Errorf("chmod %s: %w", cfg.Socket, err)
	}

	log.Info("backend listening", zap.String("socket", cfg.Socket))
	return (&http.Server{Handler: rootMux}).Serve(listener)
}

func logger() *zap.Logger {
	c := zap.NewProductionConfig()
	c.Encoding = "console"
	c.EncoderConfig.TimeKey = ""
	c.OutputPaths = []string{"stdout"}
	c.ErrorOutputPaths = []string{"stderr"}
	log, err := c.Build()
	if err != nil {
		panic(err)
	}
	return log
}
