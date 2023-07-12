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

type TargetApis struct {
	CreateRoute    httprouter.Handle
	DeleteRoute    httprouter.Handle
	CreateRedirect httprouter.Handle
	DeleteRedirect httprouter.Handle
}

func SetupTargetApis(verify mjwt.Verifier, manager *router.Manager) *TargetApis {
	r := &TargetApis{
		CreateRoute: parseJsonAndCheckOwnership[routeSource](verify, "route", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t routeSource) {
			err := manager.InsertRoute(target.Route(t))
			if err != nil {
				log.Printf("[Violet] Failed to insert route into database: %s\n", err)
				apiError(rw, http.StatusInternalServerError, "Failed to insert route into database")
				return
			}
			manager.Compile()
		}),
		DeleteRoute: parseJsonAndCheckOwnership[sourceJson](verify, "route", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t sourceJson) {
			err := manager.DeleteRoute(t.Src)
			if err != nil {
				log.Printf("[Violet] Failed to delete route from database: %s\n", err)
				apiError(rw, http.StatusInternalServerError, "Failed to delete route from database")
				return
			}
			manager.Compile()
		}),
		CreateRedirect: parseJsonAndCheckOwnership[redirectSource](verify, "redirect", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t redirectSource) {
			err := manager.InsertRedirect(target.Redirect(t))
			if err != nil {
				log.Printf("[Violet] Failed to insert redirect into database: %s\n", err)
				apiError(rw, http.StatusInternalServerError, "Failed to insert redirect into database")
				return
			}
			manager.Compile()
		}),
		DeleteRedirect: parseJsonAndCheckOwnership[sourceJson](verify, "redirect", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims, t sourceJson) {
			err := manager.DeleteRedirect(t.Src)
			if err != nil {
				log.Printf("[Violet] Failed to delete redirect from database: %s\n", err)
				apiError(rw, http.StatusInternalServerError, "Failed to delete redirect from database")
				return
			}
			manager.Compile()
		}),
	}
	return r
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
