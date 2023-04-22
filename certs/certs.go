package certs

import (
	"code.mrmelon54.com/melon/certgen"
	"crypto/tls"
	"fmt"
	"github.com/MrMelon54/violet/utils"
	"io/fs"
	"log"
	"path/filepath"
	"sync"
)

type Certs struct {
	cDir fs.FS
	kDir fs.FS
	s    *sync.RWMutex
	m    map[string]*tls.Certificate
}

func New(certDir fs.FS, keyDir fs.FS) *Certs {
	a := &Certs{
		cDir: certDir,
		kDir: keyDir,
		s:    &sync.RWMutex{},
		m:    make(map[string]*tls.Certificate),
	}
	a.Compile()
	return a
}

func (c *Certs) GetCertForDomain(domain string) *tls.Certificate {
	// safety read lock
	c.s.RLock()
	defer c.s.RUnlock()

	// lookup and return cert
	if cert, ok := c.m[domain]; ok {
		return cert
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

func (c *Certs) internalCompile(m map[string]*tls.Certificate) error {
	// try to read dir
	files, err := fs.ReadDir(c.cDir, "")
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
		keyName := name[:len(name)-len(ext)] + "key"

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
