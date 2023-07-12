package conf

import (
	"database/sql"
	"github.com/MrMelon54/mjwt"
	errorPages "github.com/MrMelon54/violet/error-pages"
	"github.com/MrMelon54/violet/favicons"
	"github.com/MrMelon54/violet/router"
	"github.com/MrMelon54/violet/utils"
)

// Conf stores the shared configuration for the API, HTTP and HTTPS servers.
type Conf struct {
	ApiListen   string // api server listen address
	HttpListen  string // http server listen address
	HttpsListen string // https server listen address
	RateLimit   uint64 // rate limit per minute
	DB          *sql.DB
	Domains     utils.DomainProvider
	Acme        utils.AcmeChallengeProvider
	Certs       utils.CertProvider
	Favicons    *favicons.Favicons
	Signer      mjwt.Verifier
	ErrorPages  *errorPages.ErrorPages
	Router      *router.Manager
}
