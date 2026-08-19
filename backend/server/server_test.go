package server

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backend/pki"
)

type fakeRestarter struct {
	restarted []string
}

func (f *fakeRestarter) RestartService(name string) error {
	f.restarted = append(f.restarted, name)
	return nil
}

func newService(t *testing.T) (*Service, *fakeRestarter) {
	t.Helper()
	dataDir := t.TempDir()
	certs := &pki.PKI{Dir: path.Join(dataDir, "pki")}
	require.NoError(t, certs.Init())

	restarter := &fakeRestarter{}
	return &Service{
		DataDir:          dataDir,
		OpenvpnDir:       path.Join(dataDir, "openvpn"),
		TemplatesDir:     path.Join("..", "..", "templates"),
		ManagementSocket: path.Join(dataDir, "openvpn.socket"),
		ServerAddress:    "device.syncloud.it",
		PKI:              certs,
		Restarter:        restarter,
	}, restarter
}

func TestApply_RendersServerConf(t *testing.T) {
	service, restarter := newService(t)

	require.NoError(t, service.Apply(DefaultSettings()))

	content, err := os.ReadFile(service.ServerConfPath())
	require.NoError(t, err)
	conf := string(content)

	assert.Contains(t, conf, "port 1194")
	assert.Contains(t, conf, "proto udp")
	assert.Contains(t, conf, "dev tun")
	assert.Contains(t, conf, "topology subnet")
	assert.Contains(t, conf, "data-ciphers "+DefaultDataCiphers)
	assert.Contains(t, conf, "crl-verify "+service.PKI.CrlPath())
	assert.Equal(t, []string{ServiceName}, restarter.restarted)
}

func TestApply_ServerConfIsOpenvpn27Clean(t *testing.T) {
	service, _ := newService(t)
	require.NoError(t, service.Apply(DefaultSettings()))

	content, err := os.ReadFile(service.ServerConfPath())
	require.NoError(t, err)
	conf := string(content)

	assert.NotContains(t, conf, "comp-lzo",
		"2.7 never compresses on send; comp-lzo must not be configured")
	assert.NotRegexp(t, `(?m)^cipher `, conf,
		"--cipher is a no-op in TLS mode in 2.7, data-ciphers replaces it")
	assert.NotRegexp(t, `(?m)^dh `, conf,
		"2.7 defaults to dh none and we ship no dhparam file")
}

func TestApply_KeepsLegacyClientsWorking(t *testing.T) {
	service, _ := newService(t)
	require.NoError(t, service.Apply(DefaultSettings()))

	content, err := os.ReadFile(service.ServerConfPath())
	require.NoError(t, err)

	assert.Contains(t, string(content), "compress migrate",
		"profiles already downloaded by users carry comp-lzo; migrate mode is what keeps them connecting")
}

func TestApply_RejectsInvalidSettings(t *testing.T) {
	service, restarter := newService(t)

	settings := DefaultSettings()
	settings.Port = 0

	assert.Error(t, service.Apply(settings))
	assert.Empty(t, restarter.restarted)
}

func TestApply_PreservesDelegatedIPv6(t *testing.T) {
	service, _ := newService(t)
	require.NoError(t, service.Apply(DefaultSettings()))
	require.NoError(t, os.WriteFile(service.ServerConfPath(),
		[]byte("port 1194\nserver-ipv6 2001:db8:3::/64\n"), 0644))

	require.NoError(t, service.Apply(DefaultSettings()))

	content, err := os.ReadFile(service.ServerConfPath())
	require.NoError(t, err)
	assert.Contains(t, string(content), "server-ipv6 2001:db8:3::/64")
}

func TestClientConfig_EmbedsCredentials(t *testing.T) {
	service, _ := newService(t)
	require.NoError(t, service.PKI.IssueClient("laptop"))

	conf, err := service.ClientConfig("laptop", DefaultSettings())
	require.NoError(t, err)

	assert.Contains(t, conf, "remote device.syncloud.it 1194")
	assert.Contains(t, conf, "proto udp")
	assert.Contains(t, conf, "remote-cert-tls server")
	assert.Equal(t, 1, strings.Count(conf, "<ca>"))
	assert.Contains(t, conf, "BEGIN CERTIFICATE")
	assert.Contains(t, conf, "BEGIN PRIVATE KEY")
	assert.NotContains(t, conf, "comp-lzo")
}

func TestClientConfig_UnknownClient(t *testing.T) {
	service, _ := newService(t)
	_, err := service.ClientConfig("nobody", DefaultSettings())
	assert.Error(t, err)
}

func TestClientConfig_RejectsTraversal(t *testing.T) {
	service, _ := newService(t)
	_, err := service.ClientConfig("../../etc/passwd", DefaultSettings())
	assert.Error(t, err)
}
