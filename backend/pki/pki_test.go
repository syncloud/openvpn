package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPKI(t *testing.T) *PKI {
	p := &PKI{Dir: path.Join(t.TempDir(), "pki")}
	require.NoError(t, p.Init())
	return p
}

func TestInit_CreatesCaAndServer(t *testing.T) {
	p := newPKI(t)

	assert.FileExists(t, p.CaCertPath())
	assert.FileExists(t, p.CaKeyPath())
	assert.FileExists(t, p.CertPath(ServerName))
	assert.FileExists(t, p.KeyPath(ServerName))
	assert.FileExists(t, p.CrlPath())
}

func TestInit_Idempotent(t *testing.T) {
	p := newPKI(t)
	before, err := os.ReadFile(p.CaCertPath())
	require.NoError(t, err)

	require.NoError(t, p.Init())

	after, err := os.ReadFile(p.CaCertPath())
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestIssueClient_ChainVerifies(t *testing.T) {
	p := newPKI(t)
	require.NoError(t, p.IssueClient("laptop"))

	cert, err := p.readCert(p.CertPath("laptop"))
	require.NoError(t, err)
	caCert, err := p.readCert(p.CaCertPath())
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	assert.NoError(t, err)
	assert.Equal(t, "laptop", cert.Subject.CommonName)
}

func TestServerCert_HasServerAuth(t *testing.T) {
	p := newPKI(t)

	cert, err := p.readCert(p.CertPath(ServerName))
	require.NoError(t, err)
	assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
}

func TestIssueClient_Duplicate(t *testing.T) {
	p := newPKI(t)
	require.NoError(t, p.IssueClient("phone"))

	assert.ErrorIs(t, p.IssueClient("phone"), ErrExists)
}

func TestIssueClient_RejectsBadNames(t *testing.T) {
	p := newPKI(t)

	for _, name := range []string{"", "../escape", "a/b", "server", "with space", ".hidden"} {
		assert.Error(t, p.IssueClient(name), name)
	}
}

func TestList_ExcludesServerIncludesClients(t *testing.T) {
	p := newPKI(t)
	require.NoError(t, p.IssueClient("one"))
	require.NoError(t, p.IssueClient("two"))

	clients, err := p.List()
	require.NoError(t, err)
	require.Len(t, clients, 2)
	assert.Equal(t, "one", clients[0].Name)
	assert.Equal(t, "two", clients[1].Name)
	assert.False(t, clients[0].Revoked)
}

func TestRevoke_MarksIndexAndCrl(t *testing.T) {
	p := newPKI(t)
	require.NoError(t, p.IssueClient("lost-phone"))

	require.NoError(t, p.Revoke("lost-phone"))

	clients, err := p.List()
	require.NoError(t, err)
	require.Len(t, clients, 1)
	assert.True(t, clients[0].Revoked)

	assert.NoFileExists(t, p.CertPath("lost-phone"))
	assert.NoFileExists(t, p.KeyPath("lost-phone"))

	crl := readCRL(t, p.CrlPath())
	require.Len(t, crl.RevokedCertificateEntries, 1)
	assert.Equal(t, clients[0].Serial, crl.RevokedCertificateEntries[0].SerialNumber.Text(16))
}

func TestRevoke_Unknown(t *testing.T) {
	p := newPKI(t)
	assert.ErrorIs(t, p.Revoke("nobody"), ErrNotFound)
}

func TestRevoke_AllowsReissueOfSameName(t *testing.T) {
	p := newPKI(t)
	require.NoError(t, p.IssueClient("reused"))
	require.NoError(t, p.Revoke("reused"))

	assert.NoError(t, p.IssueClient("reused"))
}

func TestCRL_NotShortLived(t *testing.T) {
	p := newPKI(t)

	crl := readCRL(t, p.CrlPath())
	assert.True(t, crl.NextUpdate.After(time.Now().AddDate(5, 0, 0)),
		"CRL must outlive any plausible idle period, else openvpn rejects every client")
}

func TestInit_AdoptsExistingEasyRsaCa(t *testing.T) {
	dir := path.Join(t.TempDir(), "pki")
	require.NoError(t, os.MkdirAll(path.Join(dir, "private"), 0755))
	require.NoError(t, os.MkdirAll(path.Join(dir, "issued"), 0755))

	caCert, caKey := writeEasyRsaCa(t, dir)

	p := &PKI{Dir: dir}
	require.NoError(t, p.Init())

	adopted, err := p.readCert(p.CaCertPath())
	require.NoError(t, err)
	assert.Equal(t, caCert.SerialNumber, adopted.SerialNumber,
		"existing CA must be reused, not regenerated")

	require.NoError(t, p.IssueClient("after-upgrade"))
	issued, err := p.readCert(p.CertPath("after-upgrade"))
	require.NoError(t, err)
	assert.NoError(t, issued.CheckSignatureFrom(caCert))
	assert.Equal(t, caKey.PublicKey.N, adopted.PublicKey.(*rsa.PublicKey).N)
}

func writeEasyRsaCa(t *testing.T, dir string) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(4242),
		Subject:               pkix.Name{CommonName: "ChangeMe"},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(7500 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path.Join(dir, "ca.crt"),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644))
	require.NoError(t, os.WriteFile(path.Join(dir, "private", "ca.key"),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0600))

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, key
}

func readCRL(t *testing.T, file string) *x509.RevocationList {
	t.Helper()
	content, err := os.ReadFile(file)
	require.NoError(t, err)
	block, _ := pem.Decode(content)
	require.NotNil(t, block)
	crl, err := x509.ParseRevocationList(block.Bytes)
	require.NoError(t, err)
	return crl
}
