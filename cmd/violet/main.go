package main

import (
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"github.com/MrMelon54/violet/certs"
	"github.com/MrMelon54/violet/domains"
	errorPages "github.com/MrMelon54/violet/error-pages"
	"github.com/MrMelon54/violet/favicons"
	"github.com/MrMelon54/violet/proxy"
	"github.com/MrMelon54/violet/router"
	"github.com/MrMelon54/violet/servers"
	"github.com/MrMelon54/violet/utils"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// flags - each one has a usage field lol
var (
	databasePath  = flag.String("db", "", "/path/to/database.sqlite : path to the database file")
	keyPath       = flag.String("keys", "", "/path/to/keys : path contains the keys with names matching the certificates and '.key' extensions")
	certPath      = flag.String("certs", "", "/path/to/certificates : path contains the certificates to load in armoured PEM encoding")
	selfSigned    = flag.Bool("ss", false, "enable self-signed certificate mode")
	errorPagePath = flag.String("errors", "", "/path/to/error-pages : path contains the custom error pages")
	apiListen     = flag.String("api", "127.0.0.1:8080", "address for api listening")
	httpListen    = flag.String("http", "0.0.0.0:80", "address for http listening")
	httpsListen   = flag.String("https", "0.0.0.0:443", "address for https listening")
	inkscapeCmd   = flag.String("inkscape", "inkscape", "Path to inkscape binary")
	rateLimit     = flag.Uint64("ratelimit", 300, "Rate limit (max requests per minute)")
)

func main() {
	log.Println("[Violet] Starting...")
	flag.Parse()

	if *certPath != "" {
		// create path to cert dir
		err := os.MkdirAll(*certPath, os.ModePerm)
		if err != nil {
			log.Fatalf("[Violet] Failed to create certificate path '%s' does not exist", *certPath)
		}
	}
	if *keyPath != "" {
		// create path to key dir
		err := os.MkdirAll(*keyPath, os.ModePerm)
		if err != nil {
			log.Fatalf("[Violet] Failed to create certificate key path '%s' does not exist", *keyPath)
		}
	}

	// open sqlite database
	db, err := sql.Open("sqlite3", *databasePath)
	if err != nil {
		log.Fatalf("[Violet] Failed to open database '%s'...", *databasePath)
	}

	allowedDomains := domains.New(db)                                               // load allowed domains
	acmeChallenges := utils.NewAcmeChallenge()                                      // load acme challenge store
	allowedCerts := certs.New(os.DirFS(*certPath), os.DirFS(*keyPath), *selfSigned) // load certificate manager
	reverseProxy := proxy.NewHybridTransport()                                      // load reverse proxy
	dynamicFavicons := favicons.New(db, *inkscapeCmd)                               // load dynamic favicon provider
	dynamicErrorPages := errorPages.New(os.DirFS(*errorPagePath))                   // load dynamic error page provider
	dynamicRouter := router.NewManager(db, reverseProxy)                            // load dynamic router manager

	// struct containing config for the http servers
	srvConf := &servers.Conf{
		ApiListen:   *apiListen,
		HttpListen:  *httpListen,
		HttpsListen: *httpsListen,
		RateLimit:   *rateLimit,
		DB:          db,
		Domains:     allowedDomains,
		Acme:        acmeChallenges,
		Certs:       allowedCerts,
		Favicons:    dynamicFavicons,
		Verify:      nil, // TODO: add mjwt verify support
		ErrorPages:  dynamicErrorPages,
		Router:      dynamicRouter,
	}

	var srvApi, srvHttp, srvHttps *http.Server
	if *apiListen != "" {
		srvApi = servers.NewApiServer(srvConf, utils.MultiCompilable{allowedDomains, allowedCerts, dynamicFavicons, dynamicErrorPages, dynamicRouter})
		log.Printf("[API] Starting API server on: '%s'\n", srvApi.Addr)
		go utils.RunBackgroundHttp("API", srvApi)
	}
	if *httpListen != "" {
		srvHttp = servers.NewHttpServer(srvConf)
		log.Printf("[HTTP] Starting HTTP server on: '%s'\n", srvHttp.Addr)
		go utils.RunBackgroundHttp("HTTP", srvHttp)
	}
	if *httpsListen != "" {
		srvHttps = servers.NewHttpsServer(srvConf)
		log.Printf("[HTTPS] Starting HTTPS server on: '%s'\n", srvHttps.Addr)
		go utils.RunBackgroundHttps("HTTPS", srvHttps)
	}

	// Wait for exit signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	fmt.Println()

	// Stop servers
	log.Printf("[Violet] Stopping...")
	n := time.Now()

	// close http servers
	if srvApi != nil {
		srvApi.Close()
	}
	if srvHttp != nil {
		srvHttp.Close()
	}
	if srvHttps != nil {
		srvHttps.Close()
	}

	log.Printf("[Violet] Took '%s' to shutdown\n", time.Now().Sub(n))
	log.Println("[Violet] Goodbye")
}
