package servers

import (
	"database/sql"
	"github.com/MrMelon54/violet/certs"
	"github.com/MrMelon54/violet/domains"
	errorPages "github.com/MrMelon54/violet/error-pages"
	"github.com/MrMelon54/violet/favicons"
	"github.com/mrmelon54/mjwt"
	"net/http/httputil"
)

type Conf struct {
	ApiListen   string
	HttpListen  string
	HttpsListen string
	DB          *sql.DB
	Domains     *domains.Domains
	Certs       *certs.Certs
	Favicons    *favicons.Favicons
	Verify      mjwt.Provider
	ErrorPages  *errorPages.ErrorPages
	Proxy       *httputil.ReverseProxy
}
