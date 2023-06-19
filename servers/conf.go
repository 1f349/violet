package servers

import (
	"crypto/tls"
	"database/sql"
	"github.com/MrMelon54/mjwt"
	errorPages "github.com/MrMelon54/violet/error-pages"
	"github.com/MrMelon54/violet/favicons"
	"github.com/MrMelon54/violet/router"
)

// Conf stores the shared configuration for the API, HTTP and HTTPS servers.
type Conf struct {
	ApiListen   string // api server listen address
	HttpListen  string // http server listen address
	HttpsListen string // https server listen address
	RateLimit   uint64 // rate limit per minute
	DB          *sql.DB
	Domains     DomainProvider
	Acme        AcmeChallengeProvider
	Certs       CertProvider
	Favicons    *favicons.Favicons
	Verify      mjwt.Verifier
	ErrorPages  *errorPages.ErrorPages
	Router      *router.Manager
}

type DomainProvider interface {
	IsValid(host string) bool
}

type AcmeChallengeProvider interface {
	Get(domain, key string) string
	Put(domain, key, value string)
	Delete(domain, key string)
}

type CertProvider interface {
	GetCertForDomain(domain string) *tls.Certificate
}
