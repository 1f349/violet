package certs

import (
	"crypto/tls"
	"crypto/x509/pkix"
	"fmt"
	"github.com/1f349/violet/utils"
	"github.com/MrMelon54/certgen"
	"github.com/MrMelon54/rescheduler"
	"io/fs"
	"log"
	"math/big"
	"os"
	"strings"
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
	r    *rescheduler.Rescheduler
	t    *time.Ticker
	ts   chan struct{}
}

// New creates a new cert list
func New(certDir fs.FS, keyDir fs.FS, selfCert bool) *Certs {
	c := &Certs{
		cDir: certDir,
		kDir: keyDir,
		ss:   selfCert,
		s:    &sync.RWMutex{},
		m:    make(map[string]*tls.Certificate),
		ts:   make(chan struct{}, 1),
	}

	if !selfCert {
		// the rescheduler isn't even used in self cert mode so why initialise it
		c.r = rescheduler.NewRescheduler(c.threadCompile)

		c.t = time.NewTicker(2 * time.Hour)
		go func() {
			for {
				select {
				case <-c.t.C:
					c.Compile()
				case <-c.ts:
					return
				}
			}
		}()
	} else {
		// in self-signed mode generate a CA certificate to sign other certificates
		ca, err := certgen.MakeCaTls(4096, pkix.Name{
			Country:            []string{"GB"},
			Organization:       []string{"Violet"},
			OrganizationalUnit: []string{"Development"},
			SerialNumber:       "0",
			CommonName:         fmt.Sprintf("%d.violet.test", time.Now().Unix()),
		}, big.NewInt(0), func(now time.Time) time.Time {
			return now.AddDate(10, 0, 0)
		})
		if err != nil {
			log.Fatalln("Failed to generate CA cert for self-signed mode:", err)
		}
		c.ca = ca
	}
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
		serverTls, err := certgen.MakeServerTls(c.ca, 4096, pkix.Name{
			Country:            []string{"GB"},
			Organization:       []string{domain},
			OrganizationalUnit: []string{domain},
			SerialNumber:       fmt.Sprintf("%d", sn),
			CommonName:         domain,
		}, big.NewInt(sn), func(now time.Time) time.Time {
			return now.AddDate(10, 0, 0)
		}, []string{domain}, nil)
		if err != nil {
			return nil
		}

		// save the generated leaf for loading if the domain is requested again
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

// Compile loads the certificates and keys from the directories.
//
// This method makes use of the rescheduler instead of just ignoring multiple
// calls.
func (c *Certs) Compile() {
	// don't bother compiling in self-signed mode
	if c.ss {
		return
	}
	c.r.Run()
}

func (c *Certs) Stop() {
	if c.t != nil {
		c.t.Stop()
	}
	close(c.ts)
}

func (c *Certs) threadCompile() {
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
		if !strings.HasSuffix(name, ".cert.pem") {
			continue
		}
		keyName := name[:len(name)-len("cert.pem")] + "key.pem"

		// try to read cert file
		certData, err := fs.ReadFile(c.cDir, name)
		if err != nil {
			return fmt.Errorf("failed to read cert file '%s': %w", name, err)
		}

		// try to read key file
		keyData, err := fs.ReadFile(c.kDir, keyName)
		if err != nil {
			// ignore the file if the certificate doesn't exist
			if os.IsNotExist(err) {
				continue
			}
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
