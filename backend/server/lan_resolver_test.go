package server

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resolve(t *testing.T, content string) Resolver {
	t.Helper()
	file := path.Join(t.TempDir(), "resolv.conf")
	require.NoError(t, os.WriteFile(file, []byte(content), 0644))
	resolver, err := (&LanResolver{ResolvConf: file}).Resolve()
	require.NoError(t, err)
	return resolver
}

func TestLanResolver_RouterAndDomain(t *testing.T) {
	resolver := resolve(t, "domain home.arpa\nsearch home.arpa\nnameserver 192.168.1.1\n")
	assert.Equal(t, Resolver{DNS: []string{"192.168.1.1"}, Domain: "home.arpa"}, resolver)
}

func TestLanResolver_IPv6Nameservers(t *testing.T) {
	resolver := resolve(t, "nameserver 192.168.1.1\nnameserver 2a00:23c5::1\nnameserver fe80::1%eth0\n")
	assert.Equal(t, []string{"192.168.1.1"}, resolver.DNS)
	assert.Equal(t, []string{"2a00:23c5::1"}, resolver.DNS6)
}

func TestLanResolver_SearchListUsesFirstDomain(t *testing.T) {
	resolver := resolve(t, "search lan example.com\nnameserver 10.0.0.1\n")
	assert.Equal(t, "lan", resolver.Domain)
}

func TestLanResolver_LoopbackOnlyFallsBackToPublic(t *testing.T) {
	assert.Equal(t, PublicResolver, resolve(t, "nameserver 127.0.0.11\noptions ndots:0\n"))
	assert.Equal(t, PublicResolver, resolve(t, "nameserver 127.0.0.53\nsearch lan\n"))
	assert.Equal(t, PublicResolver, resolve(t, "nameserver ::1\n"))
}

func TestLanResolver_MissingFileFallsBackToPublic(t *testing.T) {
	resolver, err := (&LanResolver{ResolvConf: path.Join(t.TempDir(), "missing")}).Resolve()
	require.NoError(t, err)
	assert.Equal(t, PublicResolver, resolver)
}

func TestLanResolver_UnreadableFileIsAnError(t *testing.T) {
	_, err := (&LanResolver{ResolvConf: t.TempDir()}).Resolve()
	assert.Error(t, err)
}
