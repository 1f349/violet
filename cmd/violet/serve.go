package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/MrMelon54/mjwt"
	"github.com/MrMelon54/violet/certs"
	"github.com/MrMelon54/violet/domains"
	errorPages "github.com/MrMelon54/violet/error-pages"
	"github.com/MrMelon54/violet/favicons"
	"github.com/MrMelon54/violet/proxy"
	"github.com/MrMelon54/violet/router"
	"github.com/MrMelon54/violet/servers"
	"github.com/MrMelon54/violet/utils"
	"github.com/google/subcommands"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type serveCmd struct{ configPath string }

func (s *serveCmd) Name() string     { return "serve" }
func (s *serveCmd) Synopsis() string { return "Serve reverse proxy server" }
func (s *serveCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&s.configPath, "conf", "", "/path/to/config.json : path to the config file")
}
func (s *serveCmd) Usage() string {
	return `serve [-conf <config file>]
  Serve reverse proxy server using information from config file
`
}

func (s *serveCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	log.Println("[Violet] Starting...")

	if s.configPath == "" {
		log.Println("[Violet] Error: config flag is missing")
		return subcommands.ExitUsageError
	}

	openConf, err := os.Open(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("[Violet] Error: missing config file")
		} else {
			log.Println("[Violet] Error: open config file: ", err)
		}
		return subcommands.ExitFailure
	}

	var conf startUpConfig
	err = json.NewDecoder(openConf).Decode(&conf)
	if err != nil {
		log.Println("[Violet] Error: invalid config file: ", err)
		return subcommands.ExitFailure
	}

	normalLoad(conf)
	return subcommands.ExitSuccess
}

func normalLoad(conf startUpConfig) {
	// the cert and key paths are useless in self-signed mode
	if !conf.SelfSigned {
		if conf.CertPath != "" {
			// create path to cert dir
			err := os.MkdirAll(conf.CertPath, os.ModePerm)
			if err != nil {
				log.Fatalf("[Violet] Failed to create certificate path '%s'", conf.CertPath)
			}
		}
		if conf.KeyPath != "" {
			// create path to key dir
			err := os.MkdirAll(conf.KeyPath, os.ModePerm)
			if err != nil {
				log.Fatalf("[Violet] Failed to create certificate key path '%s'", conf.KeyPath)
			}
		}
	}

	// errorPageDir stores an FS interface for accessing the error page directory
	var errorPageDir fs.FS
	if conf.ErrorPagePath != "" {
		errorPageDir = os.DirFS(conf.ErrorPagePath)
		err := os.MkdirAll(conf.ErrorPagePath, os.ModePerm)
		if err != nil {
			log.Fatalf("[Violet] Failed to create error page path '%s'", conf.ErrorPagePath)
		}
	}

	// load the MJWT RSA public key from a pem encoded file
	mjwtVerify, err := mjwt.NewMJwtVerifierFromFile(conf.MjwtPubKey)
	if err != nil {
		log.Fatalf("[Violet] Failed to load MJWT verifier public key from file: '%s'", conf.MjwtPubKey)
	}

	// open sqlite database
	db, err := sql.Open("sqlite3", conf.Database)
	if err != nil {
		log.Fatalf("[Violet] Failed to open database '%s'...", conf.Database)
	}

	allowedDomains := domains.New(db)                                                           // load allowed domains
	acmeChallenges := utils.NewAcmeChallenge()                                                  // load acme challenge store
	allowedCerts := certs.New(os.DirFS(conf.CertPath), os.DirFS(conf.KeyPath), conf.SelfSigned) // load certificate manager
	hybridTransport := proxy.NewHybridTransport()                                               // load reverse proxy
	dynamicFavicons := favicons.New(db, conf.InkscapeCmd)                                       // load dynamic favicon provider
	dynamicErrorPages := errorPages.New(errorPageDir)                                           // load dynamic error page provider
	dynamicRouter := router.NewManager(db, hybridTransport)                                     // load dynamic router manager

	// struct containing config for the http servers
	srvConf := &servers.Conf{
		ApiListen:   conf.Listen.Api,
		HttpListen:  conf.Listen.Http,
		HttpsListen: conf.Listen.Https,
		RateLimit:   conf.RateLimit,
		DB:          db,
		Domains:     allowedDomains,
		Acme:        acmeChallenges,
		Certs:       allowedCerts,
		Favicons:    dynamicFavicons,
		Verify:      mjwtVerify,
		ErrorPages:  dynamicErrorPages,
		Router:      dynamicRouter,
	}

	// create the compilable list and run a first time compile
	allCompilables := utils.MultiCompilable{allowedDomains, allowedCerts, dynamicFavicons, dynamicErrorPages, dynamicRouter}
	allCompilables.Compile()

	var srvApi, srvHttp, srvHttps *http.Server
	if srvConf.ApiListen != "" {
		srvApi = servers.NewApiServer(srvConf, allCompilables)
		log.Printf("[API] Starting API server on: '%s'\n", srvApi.Addr)
		go utils.RunBackgroundHttp("API", srvApi)
	}
	if srvConf.HttpListen != "" {
		srvHttp = servers.NewHttpServer(srvConf)
		log.Printf("[HTTP] Starting HTTP server on: '%s'\n", srvHttp.Addr)
		go utils.RunBackgroundHttp("HTTP", srvHttp)
	}
	if srvConf.HttpsListen != "" {
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
