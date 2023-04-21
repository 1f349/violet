package servers

import (
	"code.mrmelon54.com/melon/summer-utils/claims/auth"
	"github.com/MrMelon54/violet/utils"
	"github.com/julienschmidt/httprouter"
	"github.com/mrmelon54/mjwt"
	"log"
	"net/http"
	"time"
)

// NewApiServer creates and runs a *http.Server containing all the API endpoints for the software
//
// `/compile` - reloads all domains, routes and redirects from the configuration files
func NewApiServer(listen string, verify mjwt.Provider, compileTarget utils.MultiCompilable) *http.Server {
	r := httprouter.New()

	// Endpoint `/compile` reloads all domains, routes and redirects from the configuration files
	r.POST("/compile", func(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		// Get bearer token
		bearer := utils.GetBearer(req)
		if bearer == "" {
			utils.RespondHttpStatus(rw, http.StatusForbidden)
			return
		}

		// Read claims from mjwt
		_, b, err := mjwt.ExtractClaims[auth.AccessTokenClaims](verify, bearer)
		if err != nil {
			utils.RespondHttpStatus(rw, http.StatusForbidden)
			return
		}

		// Token must have `violet:compile` perm
		if !b.Claims.Perms.Has("violet:compile") {
			utils.RespondHttpStatus(rw, http.StatusForbidden)
			return
		}

		// Trigger the compile action
		compileTarget.Compile()
		rw.WriteHeader(http.StatusAccepted)
	})

	// Create and run http server
	s := &http.Server{
		Addr:              listen,
		Handler:           r,
		ReadTimeout:       time.Minute,
		ReadHeaderTimeout: time.Minute,
		WriteTimeout:      time.Minute,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    2500,
	}
	log.Printf("[API] Starting API server on: '%s'\n", s.Addr)
	go utils.RunBackgroundHttp("API", s)
	return s
}
