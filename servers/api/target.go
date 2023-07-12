package api

import (
	"encoding/json"
	"github.com/MrMelon54/mjwt"
	"github.com/MrMelon54/violet/router"
	"github.com/MrMelon54/violet/target"
	"github.com/MrMelon54/violet/utils"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strings"
)

func SetupTargetApis(r *httprouter.Router, verify mjwt.Verifier, manager *router.Manager) {
	// Endpoint for routes
	r.GET("/route", checkAuthWithPerm(verify, "violet:route", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims) {
		routes, err := manager.GetAllRoutes()
		if err != nil {
			apiError(rw, http.StatusInternalServerError, "Failed to get routes from database")
			return
		}
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(routes)
	}))
	r.POST("/route", parseJsonAndCheckOwnership[routeSource](verify, "route", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t routeSource) {
		err := manager.InsertRoute(target.Route(t))
		if err != nil {
			log.Printf("[Violet] Failed to insert route into database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to insert route into database")
			return
		}
		manager.Compile()
	}))
	r.DELETE("/route", parseJsonAndCheckOwnership[sourceJson](verify, "route", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t sourceJson) {
		err := manager.DeleteRoute(t.Src)
		if err != nil {
			log.Printf("[Violet] Failed to delete route from database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to delete route from database")
			return
		}
		manager.Compile()
	}))

	// Endpoint for redirects
	r.GET("/redirect", checkAuthWithPerm(verify, "violet:redirect", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims) {
		redirects, err := manager.GetAllRedirects()
		if err != nil {
			apiError(rw, http.StatusInternalServerError, "Failed to get redirects from database")
			return
		}
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(redirects)
	}))
	r.POST("/redirect", parseJsonAndCheckOwnership[redirectSource](verify, "redirect", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t redirectSource) {
		err := manager.InsertRedirect(target.Redirect(t))
		if err != nil {
			log.Printf("[Violet] Failed to insert redirect into database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to insert redirect into database")
			return
		}
		manager.Compile()
	}))
	r.DELETE("/redirect", parseJsonAndCheckOwnership[sourceJson](verify, "redirect", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t sourceJson) {
		err := manager.DeleteRedirect(t.Src)
		if err != nil {
			log.Printf("[Violet] Failed to delete redirect from database: %s\n", err)
			apiError(rw, http.StatusInternalServerError, "Failed to delete redirect from database")
			return
		}
		manager.Compile()
	}))
}

type AuthWithJsonCallback[T any] func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t T)

func parseJsonAndCheckOwnership[T sourceGetter](verify mjwt.Verifier, t string, cb AuthWithJsonCallback[T]) httprouter.Handle {
	return checkAuthWithPerm(verify, "violet:"+t, func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims) {
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
