package servers

import (
	"fmt"
	"github.com/MrMelon54/violet/domains"
	"github.com/MrMelon54/violet/utils"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func NewHttpServer(listen string, httpsPort uint16, domainCheck *domains.Domains) *http.Server {
	r := httprouter.New()
	r.GET("/.well-known/acme-challenge/{token}", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		if hostname, ok := utils.GetDomainWithoutPort(req.Host); ok {
			if !domainCheck.IsValid(req.Host) {
				http.Error(rw, fmt.Sprintf("%d %s\n", 420, "Invalid host"), 420)
				return
			}
			if tokenValue := params.ByName("token"); tokenValue != "" {
				rw.WriteHeader(http.StatusOK)
				return
			}
		}
		rw.WriteHeader(http.StatusNotFound)
	})
}
