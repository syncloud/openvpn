package installer

import (
	"bufio"
	"encoding/json"
	"os"
	"path"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

const (
	DefaultPort  = 1194
	DefaultProto = "udp"
)

type Legacy struct {
	Port  int    `json:"port"`
	Proto string `json:"proto"`
}

func MigrateLegacy(dataDir string, logger *zap.Logger) error {
	target := path.Join(dataDir, "legacy.json")
	if _, err := os.Stat(target); err == nil {
		return nil
	}

	serverConf := path.Join(dataDir, "openvpn", "server.conf")
	content, err := os.ReadFile(serverConf)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("no legacy server.conf, fresh install")
			return nil
		}
		return err
	}

	legacy := ParseLegacyServerConf(string(content))
	logger.Info("adopting legacy openvpn settings",
		zap.Int("port", legacy.Port),
		zap.String("proto", legacy.Proto))

	data, err := json.Marshal(legacy)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0644)
}

func ParseLegacyServerConf(content string) Legacy {
	legacy := Legacy{Port: DefaultPort, Proto: DefaultProto}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 || strings.HasPrefix(fields[0], "#") || strings.HasPrefix(fields[0], ";") {
			continue
		}
		switch fields[0] {
		case "port":
			if port, err := strconv.Atoi(fields[1]); err == nil && port > 0 && port < 65536 {
				legacy.Port = port
			}
		case "proto":
			proto := strings.ToLower(fields[1])
			if proto == "udp" || proto == "tcp" || proto == "udp6" || proto == "tcp6" {
				legacy.Proto = proto
			}
		}
	}
	return legacy
}
