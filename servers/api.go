package servers

import (
	"github.com/MrMelon54/mjwt"
	"github.com/MrMelon54/mjwt/auth"
	"github.com/MrMelon54/violet/utils"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"time"
)

// NewApiServer creates and runs a http server containing all the API
// endpoints for the software
//
// `/compile` - reloads all domains, routes and redirects
func NewApiServer(conf *Conf, compileTarget utils.MultiCompilable) *http.Server {
	r := httprouter.New()

	// Endpoint for compile action
	r.POST("/compile", func(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		if !hasPerms(conf.Verify, req, "violet:compile") {
			utils.RespondHttpStatus(rw, http.StatusForbidden)
			return
		}

		// Trigger the compile action
		compileTarget.Compile()
		rw.WriteHeader(http.StatusAccepted)
	})

	// Endpoint for acme-challenge
	r.PUT("/acme-challenge/:domain/:key/:value", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		if !hasPerms(conf.Verify, req, "violet:acme-challenge") {
			utils.RespondHttpStatus(rw, http.StatusForbidden)
			return
		}
		domain := params.ByName("domain")
		if !conf.Domains.IsValid(domain) {
			utils.RespondVioletError(rw, http.StatusBadRequest, "Invalid ACME challenge domain")
			return
		}
		conf.Acme.Put(domain, params.ByName("key"), params.ByName("value"))
		rw.WriteHeader(http.StatusAccepted)
	})
	r.DELETE("/acme-challenge/:domain/:key", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		if !hasPerms(conf.Verify, req, "violet:acme-challenge") {
			utils.RespondHttpStatus(rw, http.StatusForbidden)
			return
		}
		domain := params.ByName("domain")
		if !conf.Domains.IsValid(domain) {
			utils.RespondVioletError(rw, http.StatusBadRequest, "Invalid ACME challenge domain")
			return
		}
		conf.Acme.Delete(domain, params.ByName("key"))
		rw.WriteHeader(http.StatusAccepted)
	})

	// Create and run http server
	return &http.Server{
		Addr:              conf.ApiListen,
		Handler:           r,
		ReadTimeout:       time.Minute,
		ReadHeaderTimeout: time.Minute,
		WriteTimeout:      time.Minute,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    2500,
	}
}

func hasPerms(verify mjwt.Verifier, req *http.Request, perm string) bool {
	// Get bearer token
	bearer := utils.GetBearer(req)
	if bearer == "" {
		return false
	}

	// Read claims from mjwt
	_, b, err := mjwt.ExtractClaims[auth.AccessTokenClaims](verify, bearer)
	if err != nil {
		return false
	}

	// Token must have perm
	return b.Claims.Perms.Has(perm)
}
