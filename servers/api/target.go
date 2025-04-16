package api

import (
	"encoding/json"
	"github.com/1f349/mjwt"
	"github.com/1f349/violet/logger"
	"github.com/1f349/violet/router"
	"github.com/1f349/violet/target"
	"github.com/1f349/violet/utils"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
)

func SetupTargetApis(r *httprouter.Router, keyStore *mjwt.KeyStore, manager *router.Manager) {
	// Endpoint for routes
	r.GET("/route", checkAuthWithPerm(keyStore, "violet:route", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims) {
		domains := getDomainOwnershipClaims(b.Claims.Perms)

		routes, err := manager.GetAllRoutes(domains)
		if err != nil {
			logger.Logger.Infof("Failed to get routes from database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to get routes from database")
			return
		}
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(routes)
	}))
	r.POST("/route", parseJsonAndCheckOwnership[routeSource](keyStore, "route", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t routeSource) {
		err := manager.InsertRoute(target.RouteWithActive(t))
		if err != nil {
			logger.Logger.Infof("Failed to insert route into database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to insert route into database")
			return
		}
		manager.Compile()
	}))
	r.DELETE("/route", parseJsonAndCheckOwnership[sourceJson](keyStore, "route", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t sourceJson) {
		err := manager.DeleteRoute(t.Src)
		if err != nil {
			logger.Logger.Infof("Failed to delete route from database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to delete route from database")
			return
		}
		manager.Compile()
	}))

	// Endpoint for redirects
	r.GET("/redirect", checkAuthWithPerm(keyStore, "violet:redirect", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims) {
		domains := getDomainOwnershipClaims(b.Claims.Perms)

		redirects, err := manager.GetAllRedirects(domains)
		if err != nil {
			logger.Logger.Infof("Failed to get redirects from database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to get redirects from database")
			return
		}
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(redirects)
	}))
	r.POST("/redirect", parseJsonAndCheckOwnership[redirectSource](keyStore, "redirect", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t redirectSource) {
		err := manager.InsertRedirect(target.RedirectWithActive(t))
		if err != nil {
			logger.Logger.Infof("Failed to insert redirect into database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to insert redirect into database")
			return
		}
		manager.Compile()
	}))
	r.DELETE("/redirect", parseJsonAndCheckOwnership[sourceJson](keyStore, "redirect", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t sourceJson) {
		err := manager.DeleteRedirect(t.Src)
		if err != nil {
			logger.Logger.Infof("Failed to delete redirect from database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to delete redirect from database")
			return
		}
		manager.Compile()
	}))
}

type AuthWithJsonCallback[T any] func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t T)

func parseJsonAndCheckOwnership[T sourceGetter](keyStore *mjwt.KeyStore, t string, cb AuthWithJsonCallback[T]) httprouter.Handle {
	return checkAuthWithPerm(keyStore, "violet:"+t, func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims) {
		var j T
		if json.NewDecoder(req.Body).Decode(&j) != nil {
			apiError(rw, http.StatusBadRequest, "Invalid request body")
			return
		}

		// check token owns this domain
		host, _ := utils.SplitHostPath(j.GetSource())
		if strings.IndexByte(host, ':') != -1 {
			apiError(rw, http.StatusBadRequest, "Invalid route source")
			return
		}

		if !validateDomainOwnershipClaims(host, b.Claims.Perms) {
			apiError(rw, http.StatusBadRequest, "Token cannot modify the specified domain")
			return
		}

		cb(rw, req, params, b, j)
	})
}
