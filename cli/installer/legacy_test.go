package installer

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestParseLegacyServerConf_Defaults(t *testing.T) {
	legacy := ParseLegacyServerConf("dev tun\ntopology subnet\n")
	assert.Equal(t, DefaultPort, legacy.Port)
	assert.Equal(t, DefaultProto, legacy.Proto)
}

func TestParseLegacyServerConf_ReadsPortAndProto(t *testing.T) {
	legacy := ParseLegacyServerConf("port 1195\nproto tcp\ndev tun\n")
	assert.Equal(t, 1195, legacy.Port)
	assert.Equal(t, "tcp", legacy.Proto)
}

func TestParseLegacyServerConf_IgnoresComments(t *testing.T) {
	legacy := ParseLegacyServerConf("#port 9999\n;proto tcp\nport 1196\n")
	assert.Equal(t, 1196, legacy.Port)
	assert.Equal(t, DefaultProto, legacy.Proto)
}

func TestParseLegacyServerConf_IgnoresInvalidPort(t *testing.T) {
	legacy := ParseLegacyServerConf("port notanumber\n")
	assert.Equal(t, DefaultPort, legacy.Port)
}

func TestMigrateLegacy_NoServerConf(t *testing.T) {
	dataDir := t.TempDir()
	assert.NoError(t, os.MkdirAll(path.Join(dataDir, "openvpn"), 0755))

	assert.NoError(t, MigrateLegacy(dataDir, zap.NewNop()))

	_, err := os.Stat(path.Join(dataDir, "legacy.json"))
	assert.True(t, os.IsNotExist(err))
}

func TestMigrateLegacy_AdoptsExistingPort(t *testing.T) {
	dataDir := t.TempDir()
	assert.NoError(t, os.MkdirAll(path.Join(dataDir, "openvpn"), 0755))
	assert.NoError(t, os.WriteFile(
		path.Join(dataDir, "openvpn", "server.conf"),
		[]byte("port 1197\nproto udp\ncomp-lzo\n"), 0644))

	assert.NoError(t, MigrateLegacy(dataDir, zap.NewNop()))

	content, err := os.ReadFile(path.Join(dataDir, "legacy.json"))
	assert.NoError(t, err)
	var legacy Legacy
	assert.NoError(t, json.Unmarshal(content, &legacy))
	assert.Equal(t, 1197, legacy.Port)
	assert.Equal(t, "udp", legacy.Proto)
}

func TestMigrateLegacy_DoesNotOverwrite(t *testing.T) {
	dataDir := t.TempDir()
	assert.NoError(t, os.MkdirAll(path.Join(dataDir, "openvpn"), 0755))
	assert.NoError(t, os.WriteFile(path.Join(dataDir, "legacy.json"), []byte(`{"port":1,"proto":"udp"}`), 0644))
	assert.NoError(t, os.WriteFile(
		path.Join(dataDir, "openvpn", "server.conf"), []byte("port 1197\n"), 0644))

	assert.NoError(t, MigrateLegacy(dataDir, zap.NewNop()))

	content, err := os.ReadFile(path.Join(dataDir, "legacy.json"))
	assert.NoError(t, err)
	assert.Contains(t, string(content), `"port":1`)
}
