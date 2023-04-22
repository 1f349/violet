package main

import (
	"database/sql"
	_ "embed"
	"flag"
	"github.com/MrMelon54/violet/certs"
	"github.com/MrMelon54/violet/domains"
	"github.com/MrMelon54/violet/proxy"
	"github.com/MrMelon54/violet/router"
	"github.com/MrMelon54/violet/servers"
	"github.com/MrMelon54/violet/utils"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

var (
	databasePath  = flag.String("db", "", "/path/to/database.sqlite")
	keyPath       = flag.String("keys", "", "/path/to/keys : path contains the keys with names matching the certificates and '.key' extensions")
	certPath      = flag.String("certs", "", "/path/to/certificates : path contains the certificates to load in armoured PEM encoding")
	errorPagePath = flag.String("errors", "", "/path/to/error-pages : path contains the custom error pages")
	apiListen     = flag.String("api", "127.0.0.1:8080", "address for api listening")
	httpListen    = flag.String("http", "0.0.0.0:80", "address for http listening")
	httpsListen   = flag.String("https", "0.0.0.0:443", "address for https listening")
)

func main() {
	log.Println("[Violet] Starting...")

	// create paths
	err := os.MkdirAll(*certPath, os.ModePerm)
	if err != nil {
		log.Fatalf("[Violet] Failed to create certificate path '%s' does not exist", *certPath)
	}
	err = os.MkdirAll(*keyPath, os.ModePerm)
	if err != nil {
		log.Fatalf("[Violet] Failed to create certificate key path '%s' does not exist", *keyPath)
	}

	// open sqlite database
	db, err := sql.Open("sqlite3", *databasePath)
	if err != nil {
		log.Fatalf("[Violet] Failed to open database '%s'...", *databasePath)
	}

	// load allowed domains
	allowedDomains := domains.New(db)

	// load allowed certificates
	allowedCerts := certs.New(os.DirFS(*certPath), os.DirFS(*keyPath))

	// create reverse proxy and
	reverseProxy := proxy.CreateHybridReverseProxy()
	r := router.New(reverseProxy)

	srvConf := &servers.Conf{
		ApiListen:   *apiListen,
		HttpListen:  *httpListen,
		HttpsListen: *httpsListen,
		DB:          db,
		Domains:     allowedDomains,
		Certs:       allowedCerts,
		Favicons:    dynamicFavicons,
		Verify:      apiVerify,
		ErrorPages:  dynamicErrorPages,
		Proxy:       reverseProxy,
	}

	if *apiListen != "" {
		servers.NewApiServer(*apiListen, nil, utils.MultiCompilable{allowedDomains})
	}
	if *httpListen != "" {
		servers.NewHttpServer(srvConf)
	}
	if *httpsListen != "" {
		servers.NewHttpsServer(srvConf)
	}
}
