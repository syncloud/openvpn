package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path"
	"regexp"
	"time"
)

const (
	ServerName = "server"

	caValidity     = 7500 * 24 * time.Hour
	certValidity   = 3650 * 24 * time.Hour
	keyBits        = 2048
	serialBits     = 128
	organization   = "Syncloud"
	caCommonName   = "Syncloud OpenVPN CA"
	filePermPublic = 0644
	filePermSecret = 0600
)

// OpenVPN refuses every client once the CRL in --crl-verify is past its
// nextUpdate. The CRL is rewritten on every revoke and on each startup, so a
// long horizon here is what prevents an idle device locking all clients out.
const crlValidity = 3650 * 24 * time.Hour

var nameRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}$`)

var ErrExists = errors.New("certificate already exists")
var ErrNotFound = errors.New("certificate not found")

type PKI struct {
	Dir string
}

type Client struct {
	Name      string    `json:"name"`
	Serial    string    `json:"serial"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
}

func ValidName(name string) bool {
	return nameRegexp.MatchString(name) && name != ServerName
}

func (p *PKI) CaCertPath() string     { return path.Join(p.Dir, "ca.crt") }
func (p *PKI) CaKeyPath() string      { return path.Join(p.Dir, "private", "ca.key") }
func (p *PKI) CrlPath() string        { return path.Join(p.Dir, "crl.pem") }
func (p *PKI) IndexPath() string      { return path.Join(p.Dir, "index.txt") }
func (p *PKI) SerialPath() string     { return path.Join(p.Dir, "serial") }
func (p *PKI) CertPath(n string) string { return path.Join(p.Dir, "issued", n+".crt") }
func (p *PKI) KeyPath(n string) string  { return path.Join(p.Dir, "private", n+".key") }

func (p *PKI) Init() error {
	for _, dir := range []string{p.Dir, path.Join(p.Dir, "private"), path.Join(p.Dir, "issued"), path.Join(p.Dir, "reqs")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	if err := p.ensureCA(); err != nil {
		return fmt.Errorf("ensure ca: %w", err)
	}
	if err := p.ensureServer(); err != nil {
		return fmt.Errorf("ensure server cert: %w", err)
	}
	return p.WriteCRL()
}

func (p *PKI) ensureCA() error {
	if fileExists(p.CaCertPath()) && fileExists(p.CaKeyPath()) {
		_, _, err := p.loadCA()
		return err
	}

	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return err
	}
	serial, err := randomSerial()
	if err != nil {
		return err
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   caCommonName,
			Organization: []string{organization},
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(caValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}
	if err := writePem(p.CaCertPath(), "CERTIFICATE", der, filePermPublic); err != nil {
		return err
	}
	return writePem(p.CaKeyPath(), "PRIVATE KEY", mustMarshalKey(key), filePermSecret)
}

func (p *PKI) ensureServer() error {
	if cert, err := p.readCert(p.CertPath(ServerName)); err == nil {
		if time.Now().Before(cert.NotAfter) && fileExists(p.KeyPath(ServerName)) {
			return nil
		}
	}
	_, err := p.issue(ServerName, x509.ExtKeyUsageServerAuth)
	return err
}

func (p *PKI) IssueClient(name string) error {
	if !ValidName(name) {
		return fmt.Errorf("invalid name %q", name)
	}
	entries, err := ReadIndex(p.IndexPath())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name == name && !entry.Revoked() {
			return ErrExists
		}
	}
	if fileExists(p.CertPath(name)) {
		return ErrExists
	}
	_, err = p.issue(name, x509.ExtKeyUsageClientAuth)
	return err
}

func (p *PKI) issue(name string, usage x509.ExtKeyUsage) (*x509.Certificate, error) {
	caCert, caKey, err := p.loadCA()
	if err != nil {
		return nil, err
	}

	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   name,
			Organization: []string{organization},
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(certValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{usage},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	if err := writePem(p.CertPath(name), "CERTIFICATE", der, filePermPublic); err != nil {
		return nil, err
	}
	if err := writePem(p.KeyPath(name), "PRIVATE KEY", mustMarshalKey(key), filePermSecret); err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	if err := p.appendIndex(Entry{
		Status:    StatusValid,
		ExpiresAt: cert.NotAfter,
		Serial:    serial,
		Name:      name,
	}); err != nil {
		return nil, err
	}
	return cert, os.WriteFile(p.SerialPath(),
		[]byte(fmt.Sprintf("%X\n", new(big.Int).Add(serial, big.NewInt(1)))), filePermPublic)
}

func (p *PKI) appendIndex(entry Entry) error {
	entries, err := ReadIndex(p.IndexPath())
	if err != nil {
		return err
	}
	return WriteIndex(p.IndexPath(), append(entries, entry))
}

func (p *PKI) Revoke(name string) error {
	if !ValidName(name) {
		return fmt.Errorf("invalid name %q", name)
	}
	entries, err := ReadIndex(p.IndexPath())
	if err != nil {
		return err
	}

	found := false
	for i, entry := range entries {
		if entry.Name == name && !entry.Revoked() {
			entries[i].Status = StatusRevoked
			entries[i].RevokedAt = time.Now().UTC()
			found = true
		}
	}
	if !found {
		return ErrNotFound
	}
	if err := WriteIndex(p.IndexPath(), entries); err != nil {
		return err
	}

	_ = os.Remove(p.CertPath(name))
	_ = os.Remove(p.KeyPath(name))
	return p.WriteCRL()
}

func (p *PKI) List() ([]Client, error) {
	entries, err := ReadIndex(p.IndexPath())
	if err != nil {
		return nil, err
	}
	clients := make([]Client, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == ServerName {
			continue
		}
		clients = append(clients, Client{
			Name:      entry.Name,
			Serial:    entry.Serial.Text(16),
			ExpiresAt: entry.ExpiresAt,
			CreatedAt: entry.ExpiresAt.Add(-certValidity),
			Revoked:   entry.Revoked(),
		})
	}
	return clients, nil
}

func (p *PKI) WriteCRL() error {
	caCert, caKey, err := p.loadCA()
	if err != nil {
		return err
	}
	entries, err := ReadIndex(p.IndexPath())
	if err != nil {
		return err
	}

	var revoked []x509.RevocationListEntry
	for _, entry := range entries {
		if !entry.Revoked() {
			continue
		}
		revoked = append(revoked, x509.RevocationListEntry{
			SerialNumber:   entry.Serial,
			RevocationTime: entry.RevokedAt,
		})
	}

	now := time.Now()
	list := &x509.RevocationList{
		Number:                    big.NewInt(now.Unix()),
		ThisUpdate:                now.Add(-time.Hour),
		NextUpdate:                now.Add(crlValidity),
		RevokedCertificateEntries: revoked,
	}
	der, err := x509.CreateRevocationList(rand.Reader, list, caCert, caKey)
	if err != nil {
		return err
	}
	return writePem(p.CrlPath(), "X509 CRL", der, filePermPublic)
}

func (p *PKI) ReadPem(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (p *PKI) loadCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	cert, err := p.readCert(p.CaCertPath())
	if err != nil {
		return nil, nil, err
	}
	key, err := p.readKey(p.CaKeyPath())
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func (p *PKI) readCert(file string) (*x509.Certificate, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, fmt.Errorf("%s: no pem block", file)
	}
	return x509.ParseCertificate(block.Bytes)
}

func (p *PKI) readKey(file string) (*rsa.PrivateKey, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, fmt.Errorf("%s: no pem block", file)
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%s: not an rsa key", file)
	}
	return key, nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), serialBits)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, err
	}
	return serial.Add(serial, big.NewInt(1)), nil
}

func mustMarshalKey(key *rsa.PrivateKey) []byte {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		panic(err)
	}
	return der
}

func writePem(file, blockType string, der []byte, perm os.FileMode) error {
	content := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	return os.WriteFile(file, content, perm)
}

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}
