package servers

import (
	"database/sql"
	"github.com/MrMelon54/violet/certs"
	"github.com/MrMelon54/violet/domains"
	errorPages "github.com/MrMelon54/violet/error-pages"
	"github.com/MrMelon54/violet/favicons"
	"github.com/MrMelon54/violet/router"
	"github.com/mrmelon54/mjwt"
)

// Conf stores the shared configuration for the API, HTTP and HTTPS servers.
type Conf struct {
	ApiListen   string // api server listen address
	HttpListen  string // http server listen address
	HttpsListen string // https server listen address
	DB          *sql.DB
	Domains     *domains.Domains
	Certs       *certs.Certs
	Favicons    *favicons.Favicons
	Verify      mjwt.Provider
	ErrorPages  *errorPages.ErrorPages
	Router      *router.Manager
}
