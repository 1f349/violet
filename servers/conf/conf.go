package conf

import (
	"github.com/1f349/mjwt"
	"github.com/1f349/violet/database"
	errorPages "github.com/1f349/violet/error-pages"
	"github.com/1f349/violet/favicons"
	"github.com/1f349/violet/router"
	"github.com/1f349/violet/utils"
)

// Conf stores the shared configuration for the API, HTTP and HTTPS servers.
type Conf struct {
	RateLimit  uint64 // rate limit per minute
	DB         *database.Queries
	Domains    utils.DomainProvider
	Acme       utils.AcmeChallengeProvider
	Certs      utils.CertProvider
	Favicons   *favicons.Favicons
	Signer     mjwt.Verifier
	ErrorPages *errorPages.ErrorPages
	Router     *router.Manager
}
