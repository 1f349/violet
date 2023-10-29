package api

import (
	"github.com/1f349/mjwt"
	"github.com/1f349/mjwt/auth"
	"github.com/1f349/violet/utils"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type AuthClaims mjwt.BaseTypeClaims[auth.AccessTokenClaims]

type AuthCallback func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims)

// checkAuth validates the bearer token against a mjwt.Verifier and returns an
// error message or continues to the next handler
func checkAuth(verify mjwt.Verifier, cb AuthCallback) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		// Get bearer token
		bearer := utils.GetBearer(req)
		if bearer == "" {
			apiError(rw, http.StatusForbidden, "Missing bearer token")
			return
		}

		// Read claims from mjwt
		_, b, err := mjwt.ExtractClaims[auth.AccessTokenClaims](verify, bearer)
		if err != nil {
			apiError(rw, http.StatusForbidden, "Invalid token")
			return
		}

		cb(rw, req, params, AuthClaims(b))
	}
}

// checkAuthWithPerm validates the bearer token and checks if it contains a
// required permission and returns an error message or continues to the next
// handler
func checkAuthWithPerm(verify mjwt.Verifier, perm string, cb AuthCallback) httprouter.Handle {
	return checkAuth(verify, func(rw http.ResponseWriter, req *http.Request, params httprouter.Params, b AuthClaims) {
		// check perms
		if !b.Claims.Perms.Has(perm) {
			apiError(rw, http.StatusForbidden, "No permission")
			return
		}
		cb(rw, req, params, b)
	})
}
