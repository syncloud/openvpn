package pki

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

const (
	StatusValid   = "V"
	StatusRevoked = "R"
	StatusExpired = "E"

	indexTimeLayout = "060102150405Z"
)

type Entry struct {
	Status    string
	ExpiresAt time.Time
	RevokedAt time.Time
	Serial    *big.Int
	Name      string
}

func (e Entry) Revoked() bool {
	return e.Status == StatusRevoked
}

func ReadIndex(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		entry, ok := parseIndexLine(scanner.Text())
		if !ok {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

func parseIndexLine(line string) (Entry, bool) {
	if strings.TrimSpace(line) == "" {
		return Entry{}, false
	}
	fields := strings.Split(line, "\t")
	if len(fields) < 6 {
		return Entry{}, false
	}

	serial, ok := new(big.Int).SetString(strings.TrimSpace(fields[3]), 16)
	if !ok {
		return Entry{}, false
	}

	entry := Entry{
		Status: strings.TrimSpace(fields[0]),
		Serial: serial,
		Name:   commonName(fields[5]),
	}
	if expires, err := time.Parse(indexTimeLayout, strings.TrimSpace(fields[1])); err == nil {
		entry.ExpiresAt = expires
	}
	if revoked := strings.TrimSpace(fields[2]); revoked != "" {
		if at, err := time.Parse(indexTimeLayout, revoked); err == nil {
			entry.RevokedAt = at
		}
	}
	return entry, entry.Name != ""
}

func commonName(subject string) string {
	for _, part := range strings.Split(subject, "/") {
		if name, found := strings.CutPrefix(part, "CN="); found {
			return name
		}
	}
	return ""
}

func WriteIndex(path string, entries []Entry) error {
	var builder strings.Builder
	for _, entry := range entries {
		revoked := ""
		if !entry.RevokedAt.IsZero() {
			revoked = entry.RevokedAt.UTC().Format(indexTimeLayout)
		}
		builder.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\tunknown\t/CN=%s\n",
			entry.Status,
			entry.ExpiresAt.UTC().Format(indexTimeLayout),
			revoked,
			strings.ToUpper(entry.Serial.Text(16)),
			entry.Name,
		))
	}
	return os.WriteFile(path, []byte(builder.String()), 0644)
}
