package certs

import (
	"code.mrmelon54.com/melon/certgen"
	"crypto/tls"
	"crypto/x509/pkix"
	"fmt"
	"github.com/MrMelon54/violet/utils"
	"io/fs"
	"log"
	"math/big"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Certs is the certificate loader and management system.
type Certs struct {
	cDir fs.FS
	kDir fs.FS
	ss   bool
	s    *sync.RWMutex
	m    map[string]*tls.Certificate
	ca   *certgen.CertGen
	sn   atomic.Int64
}

// New creates a new cert list
func New(certDir fs.FS, keyDir fs.FS, selfCert bool) *Certs {
	c := &Certs{
		cDir: certDir,
		kDir: keyDir,
		ss:   selfCert,
		s:    &sync.RWMutex{},
		m:    make(map[string]*tls.Certificate),
	}
	if c.ss {
		ca, err := certgen.MakeCaTls(pkix.Name{
			Country:            []string{"GB"},
			Organization:       []string{"Violet"},
			OrganizationalUnit: []string{"Development"},
			SerialNumber:       "0",
			CommonName:         fmt.Sprintf("%d.violet.test", time.Now().Unix()),
		}, big.NewInt(0))
		if err != nil {
			log.Fatalln("Failed to generate CA cert for self-signed mode:", err)
		}
		c.ca = ca
	}

	// run compile to get the initial data
	c.Compile()
	return c
}

func (c *Certs) GetCertForDomain(domain string) *tls.Certificate {
	// safety read lock
	c.s.RLock()
	defer c.s.RUnlock()

	// lookup and return cert
	if cert, ok := c.m[domain]; ok {
		return cert
	}

	// if self-signed certificate is enabled then generate a certificate
	if c.ss {
		sn := c.sn.Add(1)
		serverTls, err := certgen.MakeServerTls(c.ca, pkix.Name{
			Country:            []string{"GB"},
			Organization:       []string{domain},
			OrganizationalUnit: []string{domain},
			SerialNumber:       fmt.Sprintf("%d", sn),
			CommonName:         domain,
		}, big.NewInt(sn), []string{domain}, nil)
		if err != nil {
			return nil
		}
		leaf := serverTls.GetTlsLeaf()
		c.m[domain] = &leaf
		return &leaf
	}

	// lookup and return wildcard cert
	if wildcardDomain, ok := utils.ReplaceSubdomainWithWildcard(domain); ok {
		if cert, ok := c.m[wildcardDomain]; ok {
			return cert
		}
	}

	// no cert found
	return nil
}

func (c *Certs) Compile() {
	// don't bother compiling in self-signed mode
	if c.ss {
		return
	}

	// async compile magic
	go func() {
		// new map
		certMap := make(map[string]*tls.Certificate)

		// compile map and check errors
		err := c.internalCompile(certMap)
		if err != nil {
			log.Printf("[Certs] Compile failed: %s\n", err)
			return
		}

		// lock while replacing the map
		c.s.Lock()
		c.m = certMap
		c.s.Unlock()
	}()
}

// internalCompile is a hidden internal method for loading the certificate and
// key files
func (c *Certs) internalCompile(m map[string]*tls.Certificate) error {
	if c.cDir == nil {
		return nil
	}

	// try to read dir
	files, err := fs.ReadDir(c.cDir, ".")
	if err != nil {
		return fmt.Errorf("failed to read cert dir: %w", err)
	}

	log.Printf("[Certs] Compiling lookup table for %d certificates\n", len(files))

	// find and parse certs
	for _, i := range files {
		// skip dirs
		if i.IsDir() {
			continue
		}

		// get file name and extension
		name := i.Name()
		ext := filepath.Ext(name)
		keyName := name[:len(name)-len(ext)] + ".key"

		// try to read cert file
		certData, err := fs.ReadFile(c.cDir, name)
		if err != nil {
			return fmt.Errorf("failed to read cert file '%s': %w", name, err)
		}

		// try to read key file
		keyData, err := fs.ReadFile(c.kDir, keyName)
		if err != nil {
			return fmt.Errorf("failed to read key file '%s': %w", keyName, err)
		}

		// load key pair
		pair, err := tls.X509KeyPair(certData, keyData)
		if err != nil {
			return fmt.Errorf("failed to load x509 key pair '%s + %s': %w", name, keyName, err)
		}

		// load tls leaf
		cert := &pair
		leaf := certgen.TlsLeaf(cert)

		// save in map under each dns name
		for _, j := range leaf.DNSNames {
			m[j] = cert
		}
	}

	// well no errors happened
	return nil
}
