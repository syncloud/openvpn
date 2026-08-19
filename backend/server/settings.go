package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

const (
	DefaultPort        = 1194
	DefaultProto       = "udp"
	DefaultDataCiphers = "AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305"
	DefaultAuth        = "SHA256"
	DefaultKeepalive   = "10 120"
	DefaultMaxClients  = 10
)

type Settings struct {
	Port        int    `json:"port"`
	Proto       string `json:"proto"`
	DataCiphers string `json:"data_ciphers"`
	Auth        string `json:"auth"`
	Keepalive   string `json:"keepalive"`
	MaxClients  int    `json:"max_clients"`
}

type legacySettings struct {
	Port  int    `json:"port"`
	Proto string `json:"proto"`
}

func DefaultSettings() Settings {
	return Settings{
		Port:        DefaultPort,
		Proto:       DefaultProto,
		DataCiphers: DefaultDataCiphers,
		Auth:        DefaultAuth,
		Keepalive:   DefaultKeepalive,
		MaxClients:  DefaultMaxClients,
	}
}

func (s Settings) Validate() error {
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("port out of range: %d", s.Port)
	}
	switch s.Proto {
	case "udp", "tcp", "udp6", "tcp6":
	default:
		return fmt.Errorf("unsupported proto: %s", s.Proto)
	}
	if s.MaxClients < 1 {
		return fmt.Errorf("max clients out of range: %d", s.MaxClients)
	}
	return nil
}

func SettingsPath(dataDir string) string { return path.Join(dataDir, "settings.json") }
func LegacyPath(dataDir string) string   { return path.Join(dataDir, "legacy.json") }

func LoadSettings(dataDir string) (Settings, error) {
	content, err := os.ReadFile(SettingsPath(dataDir))
	if err == nil {
		settings := DefaultSettings()
		if err := json.Unmarshal(content, &settings); err != nil {
			return Settings{}, err
		}
		return settings, nil
	}
	if !os.IsNotExist(err) {
		return Settings{}, err
	}

	settings := DefaultSettings()
	if legacy, found, err := readLegacy(dataDir); err != nil {
		return Settings{}, err
	} else if found {
		settings.Port = legacy.Port
		settings.Proto = legacy.Proto
	}
	return settings, SaveSettings(dataDir, settings)
}

func readLegacy(dataDir string) (legacySettings, bool, error) {
	content, err := os.ReadFile(LegacyPath(dataDir))
	if err != nil {
		if os.IsNotExist(err) {
			return legacySettings{}, false, nil
		}
		return legacySettings{}, false, err
	}
	var legacy legacySettings
	if err := json.Unmarshal(content, &legacy); err != nil {
		return legacySettings{}, false, err
	}
	if legacy.Port < 1 || legacy.Port > 65535 || legacy.Proto == "" {
		return legacySettings{}, false, nil
	}
	return legacy, true, nil
}

func SaveSettings(dataDir string, settings Settings) error {
	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(SettingsPath(dataDir), content, 0644)
}
