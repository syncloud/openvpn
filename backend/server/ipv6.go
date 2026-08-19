package server

import (
	"os"
	"regexp"
	"strings"
)

// bin/prefix_delegation.sh rewrites the server-ipv6 line in place when the ISP
// delegates a prefix over DHCPv6. Re-rendering from the template would drop
// that prefix, so it is carried across every regeneration.
var (
	activeIPv6   = regexp.MustCompile(`(?m)^\s*server-ipv6\s+(\S+)\s*$`)
	templateIPv6 = regexp.MustCompile(`(?m)^\s*#\s*server-ipv6\s*$`)
)

func DelegatedIPv6Prefix(serverConf string) string {
	match := activeIPv6.FindStringSubmatch(serverConf)
	if match == nil {
		return ""
	}
	return match[1]
}

func PreserveIPv6(existing, rendered string) string {
	prefix := DelegatedIPv6Prefix(existing)
	if prefix == "" {
		return rendered
	}
	return templateIPv6.ReplaceAllString(rendered, "server-ipv6 "+prefix)
}

func writePreservingIPv6(file, rendered string) error {
	existing, err := os.ReadFile(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		existing = nil
	}
	content := PreserveIPv6(string(existing), rendered)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(file, []byte(content), 0644)
}
