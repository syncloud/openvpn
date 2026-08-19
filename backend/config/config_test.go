package config

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	file := path.Join(t.TempDir(), "backend.env")
	require.NoError(t, os.WriteFile(file, []byte(
		"# generated\n"+
			"APP_DOMAIN=openvpn.example.com\n"+
			"APP_URL=https://openvpn.example.com\n"+
			"OIDC_CLIENT_ID=openvpn\n"+
			"OIDC_CLIENT_SECRET=\"quoted secret\"\n"+
			"OIDC_AUTH_BASE_URL=https://auth.example.com\n"+
			"OIDC_REDIRECT_URI=https://openvpn.example.com/auth/callback\n"+
			"SESSION_SECRET=abc123\n"+
			"SOCKET=/var/snap/openvpn/current/backend.sock\n"+
			"DATA_DIR=/var/snap/openvpn/current\n"+
			"APP_DIR=/snap/openvpn/current\n"+
			"PKI_DIR=/var/snap/openvpn/current/pki\n"+
			"OPENVPN_DIR=/var/snap/openvpn/current/openvpn\n"+
			"MANAGEMENT_SOCKET=/var/snap/openvpn/current/openvpn.socket\n"+
			"SERVER_ADDRESS=device.example.com\n"+
			"PLATFORM_CA=/var/snap/platform/current/syncloud.ca.crt\n"+
			"\n"), 0644))

	cfg, err := Load(file)
	require.NoError(t, err)

	assert.Equal(t, "openvpn.example.com", cfg.AppDomain)
	assert.Equal(t, "quoted secret", cfg.OIDCClientSecret)
	assert.Equal(t, "https://auth.example.com", cfg.OIDCAuthBaseURL)
	assert.Equal(t, "abc123", cfg.SessionSecret)
	assert.Equal(t, "device.example.com", cfg.ServerAddress)
	assert.Equal(t, "/var/snap/openvpn/current/pki", cfg.PkiDir)
	assert.Equal(t, "/var/snap/platform/current/syncloud.ca.crt", cfg.PlatformCA)
}

func TestLoad_Missing(t *testing.T) {
	_, err := Load(path.Join(t.TempDir(), "absent.env"))
	assert.Error(t, err)
}
