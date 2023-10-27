package api

import (
	"encoding/json"
	"github.com/1f349/violet/servers/conf"
	"github.com/1f349/violet/utils"
	"github.com/MrMelon54/mjwt"
	"github.com/MrMelon54/mjwt/claims"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"time"
)

// NewApiServer creates and runs a http server containing all the API
// endpoints for the software
//
// `/compile` - reloads all domains, routes and redirects
func NewApiServer(conf *conf.Conf, compileTarget utils.MultiCompilable) *http.Server {
	r := httprouter.New()

	r.GET("/", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		http.Error(rw, "Violet API Endpoint", http.StatusOK)
	})

	// Endpoint for compile action
	r.POST("/compile", checkAuthWithPerm(conf.Signer, "violet:compile", func(rw http.ResponseWriter, req *http.Request, _ httprouter.Params, b AuthClaims) {
		// Trigger the compile action
		compileTarget.Compile()
		rw.WriteHeader(http.StatusAccepted)
	}))

	// Endpoint for domains
	domainFunc := domainManage(conf.Signer, conf.Domains)
	r.PUT("/domain/:domain", domainFunc)
	r.DELETE("/domain/:domain", domainFunc)

	SetupTargetApis(r, conf.Signer, conf.Router)

	// Endpoint for acme-challenge
	acmeChallengeFunc := acmeChallengeManage(conf.Signer, conf.Domains, conf.Acme)
	r.PUT("/acme-challenge/:domain/:key/:value", acmeChallengeFunc)
	r.DELETE("/acme-challenge/:domain/:key", acmeChallengeFunc)

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

// apiError outputs a generic JSON error message
func apiError(rw http.ResponseWriter, code int, m string) {
	rw.WriteHeader(code)
	_ = json.NewEncoder(rw).Encode(map[string]string{
		"error": m,
	})
}

func domainManage(verify mjwt.Verifier, domains utils.DomainProvider) httprouter.Handle {
	return checkAuthWithPerm(verify, "violet:domains", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims) {
		// add domain with active state
		domains.Put(params.ByName("domain"), req.Method == http.MethodPut)
		domains.Compile()
		rw.WriteHeader(http.StatusAccepted)
	})
}

func acmeChallengeManage(verify mjwt.Verifier, domains utils.DomainProvider, acme utils.AcmeChallengeProvider) httprouter.Handle {
	return checkAuthWithPerm(verify, "violet:acme-challenge", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims) {
		domain := params.ByName("domain")
		if !domains.IsValid(domain) {
			utils.RespondVioletError(rw, http.StatusBadRequest, "Invalid ACME challenge domain")
			return
		}
		if req.Method == http.MethodPut {
			acme.Put(domain, params.ByName("key"), params.ByName("value"))
		} else {
			acme.Delete(domain, params.ByName("key"))
		}
		rw.WriteHeader(http.StatusAccepted)
	})
}

// getDomainOwnershipClaims returns the domains marked as owned from PermStorage,
// they match `domain:owns=<fqdn>` where fqdn will be returned
func getDomainOwnershipClaims(perms *claims.PermStorage) []string {
	a := perms.Search("domain:owns=")
	for i := range a {
		a[i] = a[i][len("domain:owns="):]
	}
	return a
}

// validateDomainOwnershipClaims validates if the claims contain the
// `domain:owns=<fqdn>` field with the matching top level domain
func validateDomainOwnershipClaims(a string, perms *claims.PermStorage) bool {
	if fqdn, ok := utils.GetTopFqdn(a); ok {
		if perms.Has("domain:owns=" + fqdn) {
			return true
		}
	}
	return false
}
