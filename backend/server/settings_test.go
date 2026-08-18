package server

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSettings_FreshInstallDefaults(t *testing.T) {
	dataDir := t.TempDir()

	settings, err := LoadSettings(dataDir)
	require.NoError(t, err)

	assert.Equal(t, DefaultPort, settings.Port)
	assert.Equal(t, DefaultProto, settings.Proto)
	assert.Equal(t, DefaultDataCiphers, settings.DataCiphers)
	assert.FileExists(t, SettingsPath(dataDir))
}

func TestLoadSettings_SeedsFromLegacyPort(t *testing.T) {
	dataDir := t.TempDir()
	require.NoError(t, os.WriteFile(LegacyPath(dataDir), []byte(`{"port":1197,"proto":"tcp"}`), 0644))

	settings, err := LoadSettings(dataDir)
	require.NoError(t, err)

	assert.Equal(t, 1197, settings.Port, "an upgraded device must keep the port its clients already point at")
	assert.Equal(t, "tcp", settings.Proto)
	assert.Equal(t, DefaultDataCiphers, settings.DataCiphers)
}

func TestLoadSettings_ExistingWins(t *testing.T) {
	dataDir := t.TempDir()
	require.NoError(t, os.WriteFile(LegacyPath(dataDir), []byte(`{"port":1197,"proto":"tcp"}`), 0644))
	require.NoError(t, SaveSettings(dataDir, Settings{Port: 1200, Proto: "udp", MaxClients: 5}))

	settings, err := LoadSettings(dataDir)
	require.NoError(t, err)
	assert.Equal(t, 1200, settings.Port)
}

func TestLoadSettings_IgnoresBrokenLegacy(t *testing.T) {
	dataDir := t.TempDir()
	require.NoError(t, os.WriteFile(LegacyPath(dataDir), []byte(`{"port":0,"proto":""}`), 0644))

	settings, err := LoadSettings(dataDir)
	require.NoError(t, err)
	assert.Equal(t, DefaultPort, settings.Port)
}

func TestSettingsRoundTrip(t *testing.T) {
	dataDir := t.TempDir()
	require.NoError(t, SaveSettings(dataDir, Settings{
		Port: 1195, Proto: "udp", DataCiphers: "AES-128-GCM",
		Auth: "SHA256", Keepalive: "5 60", MaxClients: 3,
	}))

	settings, err := LoadSettings(dataDir)
	require.NoError(t, err)
	assert.Equal(t, 1195, settings.Port)
	assert.Equal(t, "AES-128-GCM", settings.DataCiphers)
	assert.Equal(t, 3, settings.MaxClients)
}

func TestValidate(t *testing.T) {
	valid := DefaultSettings()
	assert.NoError(t, valid.Validate())

	for _, broken := range []Settings{
		{Port: 0, Proto: "udp", MaxClients: 1},
		{Port: 70000, Proto: "udp", MaxClients: 1},
		{Port: 1194, Proto: "sctp", MaxClients: 1},
		{Port: 1194, Proto: "udp", MaxClients: 0},
	} {
		assert.Error(t, broken.Validate())
	}
}

func TestSettingsPath(t *testing.T) {
	assert.Equal(t, path.Join("/data", "settings.json"), SettingsPath("/data"))
}
