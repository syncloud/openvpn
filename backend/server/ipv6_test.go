package server

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const renderedTemplate = "dev tun\ntopology subnet\nserver 10.8.0.0 255.255.255.0\n#server-ipv6\nverb 3\n"

func TestDelegatedIPv6Prefix(t *testing.T) {
	assert.Equal(t, "2001:db8::/64",
		DelegatedIPv6Prefix("dev tun\nserver-ipv6 2001:db8::/64\nverb 3\n"))
}

func TestDelegatedIPv6Prefix_Commented(t *testing.T) {
	assert.Equal(t, "", DelegatedIPv6Prefix("dev tun\n#server-ipv6\nverb 3\n"))
}

func TestDelegatedIPv6Prefix_Absent(t *testing.T) {
	assert.Equal(t, "", DelegatedIPv6Prefix("dev tun\nverb 3\n"))
}

func TestPreserveIPv6_CarriesDelegatedPrefix(t *testing.T) {
	existing := "dev tun\nserver-ipv6 2001:db8:1::/64\nverb 3\n"

	result := PreserveIPv6(existing, renderedTemplate)

	assert.Contains(t, result, "server-ipv6 2001:db8:1::/64")
	assert.NotContains(t, result, "#server-ipv6")
}

func TestPreserveIPv6_NoPrefixLeavesPlaceholder(t *testing.T) {
	result := PreserveIPv6("dev tun\n#server-ipv6\n", renderedTemplate)
	assert.Contains(t, result, "#server-ipv6")
}

func TestWritePreservingIPv6_FreshFile(t *testing.T) {
	file := path.Join(t.TempDir(), "server.conf")

	require.NoError(t, writePreservingIPv6(file, renderedTemplate))

	content, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Contains(t, string(content), "#server-ipv6")
}

func TestWritePreservingIPv6_KeepsPrefixAcrossRerender(t *testing.T) {
	file := path.Join(t.TempDir(), "server.conf")
	require.NoError(t, os.WriteFile(file,
		[]byte("dev tun\nserver-ipv6 2001:db8:2::/64\nverb 3\n"), 0644))

	require.NoError(t, writePreservingIPv6(file, renderedTemplate))

	content, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Contains(t, string(content), "server-ipv6 2001:db8:2::/64")
}
