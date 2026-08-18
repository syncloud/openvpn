package pki

import (
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadIndex_EasyRsaFormat(t *testing.T) {
	file := path.Join(t.TempDir(), "index.txt")
	require.NoError(t, os.WriteFile(file, []byte(
		"V\t340101000000Z\t\t01\tunknown\t/CN=server\n"+
			"V\t340101000000Z\t\t02\tunknown\t/CN=laptop\n"+
			"R\t340101000000Z\t260101000000Z\t03\tunknown\t/CN=old-phone\n"), 0644))

	entries, err := ReadIndex(file)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	assert.Equal(t, "server", entries[0].Name)
	assert.Equal(t, big.NewInt(1), entries[0].Serial)
	assert.False(t, entries[0].Revoked())

	assert.Equal(t, "laptop", entries[1].Name)
	assert.True(t, entries[2].Revoked())
	assert.Equal(t, "old-phone", entries[2].Name)
	assert.False(t, entries[2].RevokedAt.IsZero())
}

func TestReadIndex_MultiFieldSubject(t *testing.T) {
	file := path.Join(t.TempDir(), "index.txt")
	require.NoError(t, os.WriteFile(file,
		[]byte("V\t340101000000Z\t\t0A\tunknown\t/C=US/O=Syncloud/CN=phone\n"), 0644))

	entries, err := ReadIndex(file)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "phone", entries[0].Name)
	assert.Equal(t, big.NewInt(10), entries[0].Serial)
}

func TestReadIndex_Missing(t *testing.T) {
	entries, err := ReadIndex(path.Join(t.TempDir(), "absent"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestReadIndex_SkipsMalformed(t *testing.T) {
	file := path.Join(t.TempDir(), "index.txt")
	require.NoError(t, os.WriteFile(file, []byte(
		"garbage\n\nV\t340101000000Z\t\tZZ\tunknown\t/CN=bad-serial\n"+
			"V\t340101000000Z\t\t04\tunknown\t/CN=good\n"), 0644))

	entries, err := ReadIndex(file)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "good", entries[0].Name)
}

func TestWriteIndex_RoundTrip(t *testing.T) {
	file := path.Join(t.TempDir(), "index.txt")
	expires := time.Date(2034, 1, 1, 0, 0, 0, 0, time.UTC)
	revoked := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	require.NoError(t, WriteIndex(file, []Entry{
		{Status: StatusValid, ExpiresAt: expires, Serial: big.NewInt(255), Name: "keep"},
		{Status: StatusRevoked, ExpiresAt: expires, RevokedAt: revoked, Serial: big.NewInt(256), Name: "gone"},
	}))

	entries, err := ReadIndex(file)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, "keep", entries[0].Name)
	assert.Equal(t, big.NewInt(255), entries[0].Serial)
	assert.Equal(t, expires, entries[0].ExpiresAt)

	assert.True(t, entries[1].Revoked())
	assert.Equal(t, revoked, entries[1].RevokedAt)
}
