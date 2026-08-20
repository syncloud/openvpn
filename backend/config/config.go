package config

import (
	"os"
	"strings"
)

type Config struct {
	AppDomain        string
	AppUrl           string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCAuthBaseURL  string
	OIDCRedirectURI  string
	SessionSecret    string
	Socket           string
	DataDir          string
	AppDir           string
	PkiDir           string
	OpenvpnDir       string
	ManagementSocket string
	ServerAddress    string
	PlatformCA       string
	AuthSocket       string
}

func Load(path string) (*Config, error) {
	values, err := loadKV(path)
	if err != nil {
		return nil, err
	}
	return &Config{
		AppDomain:        values["APP_DOMAIN"],
		AppUrl:           values["APP_URL"],
		OIDCClientID:     values["OIDC_CLIENT_ID"],
		OIDCClientSecret: values["OIDC_CLIENT_SECRET"],
		OIDCAuthBaseURL:  values["OIDC_AUTH_BASE_URL"],
		OIDCRedirectURI:  values["OIDC_REDIRECT_URI"],
		SessionSecret:    values["SESSION_SECRET"],
		Socket:           values["SOCKET"],
		DataDir:          values["DATA_DIR"],
		AppDir:           values["APP_DIR"],
		PkiDir:           values["PKI_DIR"],
		OpenvpnDir:       values["OPENVPN_DIR"],
		ManagementSocket: values["MANAGEMENT_SOCKET"],
		ServerAddress:    values["SERVER_ADDRESS"],
		PlatformCA:       values["PLATFORM_CA"],
		AuthSocket:       values["AUTH_LOCAL_SOCKET"],
	}, nil
}

func loadKV(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return values, nil
}
