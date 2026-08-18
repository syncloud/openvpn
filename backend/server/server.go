package server

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"text/template"

	"backend/pki"
)

const ServiceName = "openvpn.server"

type ServiceRestarter interface {
	RestartService(name string) error
}

type serverVariables struct {
	ManagementSocket    string
	Port                int
	Proto               string
	CaCert              string
	ServerCert          string
	ServerKey           string
	Crl                 string
	DataCiphers         string
	Auth                string
	IfconfigPoolPersist string
	Keepalive           string
	MaxClients          int
}

type clientVariables struct {
	Proto         string
	ServerAddress string
	Port          int
	DataCiphers   string
	Auth          string
	Ca            string
	Cert          string
	Key           string
}

type Service struct {
	DataDir          string
	OpenvpnDir       string
	TemplatesDir     string
	ManagementSocket string
	ServerAddress    string
	PKI              *pki.PKI
	Restarter        ServiceRestarter
}

func (s *Service) ServerConfPath() string {
	return path.Join(s.OpenvpnDir, "server.conf")
}

func (s *Service) Render(settings Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.OpenvpnDir, 0755); err != nil {
		return err
	}
	if err := SaveSettings(s.DataDir, settings); err != nil {
		return err
	}
	return s.renderServerConf(settings)
}

func (s *Service) Apply(settings Settings) error {
	if err := s.Render(settings); err != nil {
		return err
	}
	if s.Restarter == nil {
		return nil
	}
	return s.Restarter.RestartService(ServiceName)
}

func (s *Service) renderServerConf(settings Settings) error {
	rendered, err := s.render("openvpn-server.conf.tpl", serverVariables{
		ManagementSocket:    s.ManagementSocket,
		Port:                settings.Port,
		Proto:               settings.Proto,
		CaCert:              s.PKI.CaCertPath(),
		ServerCert:          s.PKI.CertPath(pki.ServerName),
		ServerKey:           s.PKI.KeyPath(pki.ServerName),
		Crl:                 s.PKI.CrlPath(),
		DataCiphers:         settings.DataCiphers,
		Auth:                settings.Auth,
		IfconfigPoolPersist: path.Join(s.OpenvpnDir, "ipp.txt"),
		Keepalive:           settings.Keepalive,
		MaxClients:          settings.MaxClients,
	})
	if err != nil {
		return err
	}
	return writePreservingIPv6(s.ServerConfPath(), rendered)
}

func (s *Service) ClientConfig(name string, settings Settings) (string, error) {
	if !pki.ValidName(name) {
		return "", fmt.Errorf("invalid name %q", name)
	}
	ca, err := os.ReadFile(s.PKI.CaCertPath())
	if err != nil {
		return "", err
	}
	cert, err := os.ReadFile(s.PKI.CertPath(name))
	if err != nil {
		return "", err
	}
	key, err := os.ReadFile(s.PKI.KeyPath(name))
	if err != nil {
		return "", err
	}
	return s.render("openvpn-client.conf.tpl", clientVariables{
		Proto:         settings.Proto,
		ServerAddress: s.ServerAddress,
		Port:          settings.Port,
		DataCiphers:   settings.DataCiphers,
		Auth:          settings.Auth,
		Ca:            string(ca),
		Cert:          string(cert),
		Key:           string(key),
	})
}

func (s *Service) render(name string, variables any) (string, error) {
	file := path.Join(s.TemplatesDir, name)
	parsed, err := template.ParseFiles(file)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", file, err)
	}
	var buffer bytes.Buffer
	if err := parsed.Execute(&buffer, variables); err != nil {
		return "", fmt.Errorf("render %s: %w", file, err)
	}
	return buffer.String(), nil
}
