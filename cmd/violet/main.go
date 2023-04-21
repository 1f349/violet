package main

import (
	"database/sql"
	_ "embed"
	"errors"
	"flag"
	"github.com/MrMelon54/violet/domains"
	"github.com/MrMelon54/violet/proxy"
	"github.com/MrMelon54/violet/router"
	"github.com/MrMelon54/violet/servers"
	"github.com/MrMelon54/violet/utils"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

//go:embed init.sql
var initSql string

var (
	databasePath = flag.String("db", "", "/path/to/database.sqlite")
	certPath     = flag.String("cert", "", "/path/to/certificates")
	apiListen    = flag.String("api", "127.0.0.1:8080", "address for api listening")
	httpListen   = flag.String("http", "0.0.0.0:80", "address for http listening")
	httpsListen  = flag.String("https", "0.0.0.0:443", "address for https listening")
)

func main() {
	log.Println("[Violet] Starting...")

	_, err := os.Stat(*certPath)
	if errors.Is(err, os.ErrNotExist) {
		log.Fatalf("[Violet] Certificate path '%s' does not exists", *certPath)
	}

	_, err = os.Stat(*databasePath)
	dbExists := !errors.Is(err, os.ErrNotExist)

	db, err := sql.Open("sqlite3", *databasePath)
	if err != nil {
		log.Fatalf("[Violet] Failed to open database '%s'...", *databasePath)
	}

	if !dbExists {
		log.Println("[Violet] Creating new database and running init.sql")
		_, err = db.Exec(initSql)
		if err != nil {
			log.Fatalf("[Violet] Failed to run init.sql")
		}
	}

	allowedDomains := domains.New()
	reverseProxy := proxy.CreateHybridReverseProxy()
	r := router.New(reverseProxy)

	servers.NewApiServer(*apiListen, nil, utils.MultiCompilable{})
	servers.NewHttpServer(*httpListen, 0, allowedDomains)
}
