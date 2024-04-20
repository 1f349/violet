package certs

import (
	"crypto/x509/pkix"
	"fmt"
	"github.com/mrmelon54/certgen"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"testing/fstest"
	"time"
)

func TestCertsNew_Lookup(t *testing.T) {
	// The following code basically copies the self-signed logic from the Certs
	// type to test that certificate files can be found and read correctly. This
	// uses a MapFS for performance during tests.

	ca, err := certgen.MakeCaTls(2048, pkix.Name{
		Country:            []string{"GB"},
		Organization:       []string{"Violet"},
		OrganizationalUnit: []string{"Development"},
		SerialNumber:       "0",
		CommonName:         fmt.Sprintf("%d.violet.test", time.Now().Unix()),
	}, big.NewInt(0), func(now time.Time) time.Time {
		return now.AddDate(10, 0, 0)
	})
	assert.NoError(t, err)

	domain := "example.com"
	sn := int64(1)
	serverTls, err := certgen.MakeServerTls(ca, 2048, pkix.Name{
		Country:            []string{"GB"},
		Organization:       []string{domain},
		OrganizationalUnit: []string{domain},
		SerialNumber:       fmt.Sprintf("%d", sn),
		CommonName:         domain,
	}, big.NewInt(sn), func(now time.Time) time.Time {
		return now.AddDate(10, 0, 0)
	}, []string{domain}, nil)
	assert.NoError(t, err)

	certDir := fstest.MapFS{
		"example.com.cert.pem": {
			Data: serverTls.GetCertPem(),
		},
	}

	keyDir := fstest.MapFS{
		"example.com.key.pem": {
			Data: serverTls.GetKeyPem(),
		},
	}

	certs := New(certDir, keyDir, false)
	assert.NoError(t, certs.internalCompile(certs.m))
	cc := certs.GetCertForDomain("example.com")
	leaf := certgen.TlsLeaf(cc)
	assert.Equal(t, []string{"example.com"}, leaf.DNSNames)

	// this cert doesn't exist
	assert.Nil(t, certs.GetCertForDomain("notexample.com"))
}

func TestCertsNew_SelfSigned(t *testing.T) {
	if testing.Short() {
		return
	}

	certs := New(nil, nil, true)
	cc := certs.GetCertForDomain("example.com")
	leaf := certgen.TlsLeaf(cc)
	assert.Equal(t, []string{"example.com"}, leaf.DNSNames)

	cc2 := certs.GetCertForDomain("notexample.com")
	leaf2 := certgen.TlsLeaf(cc2)
	assert.Equal(t, []string{"notexample.com"}, leaf2.DNSNames)
}
