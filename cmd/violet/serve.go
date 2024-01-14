package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"github.com/1f349/mjwt"
	"github.com/1f349/violet/certs"
	"github.com/1f349/violet/domains"
	errorPages "github.com/1f349/violet/error-pages"
	"github.com/1f349/violet/favicons"
	"github.com/1f349/violet/proxy"
	"github.com/1f349/violet/proxy/websocket"
	"github.com/1f349/violet/router"
	"github.com/1f349/violet/servers"
	"github.com/1f349/violet/servers/api"
	"github.com/1f349/violet/servers/conf"
	"github.com/1f349/violet/utils"
	"github.com/MrMelon54/exit-reload"
	"github.com/google/subcommands"
	"io/fs"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/pprof"
)

type serveCmd struct {
	configPath string
	cpuprofile string
}

func (s *serveCmd) Name() string     { return "serve" }
func (s *serveCmd) Synopsis() string { return "Serve reverse proxy server" }
func (s *serveCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&s.configPath, "conf", "", "/path/to/config.json : path to the config file")
	f.StringVar(&s.cpuprofile, "cpuprofile", "", "write cpu profile to file")
}
func (s *serveCmd) Usage() string {
	return `serve [-conf <config file>] [-cpuprofile <profile file>]
  Serve reverse proxy server using information from config file
`
}

func (s *serveCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	log.Println("[Violet] Starting...")

	// Enable cpu profiling
	if s.cpuprofile != "" {
		f, err := os.Create(s.cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("[Violet] CPU profiling enabled, writing to '%s'\n", s.cpuprofile)
		_ = pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

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

	var config startUpConfig
	err = json.NewDecoder(openConf).Decode(&config)
	if err != nil {
		log.Println("[Violet] Error: invalid config file: ", err)
		return subcommands.ExitFailure
	}

	// working directory is the parent of the config file
	wd := filepath.Dir(s.configPath)
	normalLoad(config, wd)
	return subcommands.ExitSuccess
}

func normalLoad(startUp startUpConfig, wd string) {
	// the cert and key paths are useless in self-signed mode
	if !startUp.SelfSigned {
		// create path to cert dir
		err := os.MkdirAll(filepath.Join(wd, "certs"), os.ModePerm)
		if err != nil {
			log.Fatal("[Violet] Failed to create certificate path")
		}
		// create path to key dir
		err = os.MkdirAll(filepath.Join(wd, "keys"), os.ModePerm)
		if err != nil {
			log.Fatal("[Violet] Failed to create certificate key path")
		}
	}

	// errorPageDir stores an FS interface for accessing the error page directory
	var errorPageDir fs.FS
	if startUp.ErrorPagePath != "" {
		errorPageDir = os.DirFS(startUp.ErrorPagePath)
		err := os.MkdirAll(startUp.ErrorPagePath, os.ModePerm)
		if err != nil {
			log.Fatalf("[Violet] Failed to create error page path '%s'", startUp.ErrorPagePath)
		}
	}

	// load the MJWT RSA public key from a pem encoded file
	mJwtVerify, err := mjwt.NewMJwtVerifierFromFile(filepath.Join(wd, "signer.public.pem"))
	if err != nil {
		log.Fatalf("[Violet] Failed to load MJWT verifier public key from file '%s': %s", filepath.Join(wd, "signer.public.pem"), err)
	}

	// open sqlite database
	db, err := sql.Open("sqlite3", filepath.Join(wd, "violet.db.sqlite"))
	if err != nil {
		log.Fatal("[Violet] Failed to open database")
	}

	certDir := os.DirFS(filepath.Join(wd, "certs"))
	keyDir := os.DirFS(filepath.Join(wd, "keys"))

	ws := websocket.NewServer()
	allowedDomains := domains.New(db)                              // load allowed domains
	acmeChallenges := utils.NewAcmeChallenge()                     // load acme challenge store
	allowedCerts := certs.New(certDir, keyDir, startUp.SelfSigned) // load certificate manager
	hybridTransport := proxy.NewHybridTransport(ws)                // load reverse proxy
	dynamicFavicons := favicons.New(db, startUp.InkscapeCmd)       // load dynamic favicon provider
	dynamicErrorPages := errorPages.New(errorPageDir)              // load dynamic error page provider
	dynamicRouter := router.NewManager(db, hybridTransport)        // load dynamic router manager

	// struct containing config for the http servers
	srvConf := &conf.Conf{
		ApiListen:   startUp.Listen.Api,
		HttpListen:  startUp.Listen.Http,
		HttpsListen: startUp.Listen.Https,
		RateLimit:   startUp.RateLimit,
		DB:          db,
		Domains:     allowedDomains,
		Acme:        acmeChallenges,
		Certs:       allowedCerts,
		Favicons:    dynamicFavicons,
		Signer:      mJwtVerify,
		ErrorPages:  dynamicErrorPages,
		Router:      dynamicRouter,
	}

	// create the compilable list and run a first time compile
	allCompilables := utils.MultiCompilable{allowedDomains, allowedCerts, dynamicFavicons, dynamicErrorPages, dynamicRouter}
	allCompilables.Compile()

	var srvApi, srvHttp, srvHttps *http.Server
	if srvConf.ApiListen != "" {
		srvApi = api.NewApiServer(srvConf, allCompilables)
		srvApi.SetKeepAlivesEnabled(false)
		log.Printf("[API] Starting API server on: '%s'\n", srvApi.Addr)
		go utils.RunBackgroundHttp("API", srvApi)
	}
	if srvConf.HttpListen != "" {
		srvHttp = servers.NewHttpServer(srvConf)
		srvHttp.SetKeepAlivesEnabled(false)
		log.Printf("[HTTP] Starting HTTP server on: '%s'\n", srvHttp.Addr)
		go utils.RunBackgroundHttp("HTTP", srvHttp)
	}
	if srvConf.HttpsListen != "" {
		srvHttps = servers.NewHttpsServer(srvConf)
		srvHttps.SetKeepAlivesEnabled(false)
		log.Printf("[HTTPS] Starting HTTPS server on: '%s'\n", srvHttps.Addr)
		go utils.RunBackgroundHttps("HTTPS", srvHttps)
	}

	go func() {
		log.Println(http.ListenAndServe("localhost:6600", nil))
	}()

	exit_reload.ExitReload("Violet", func() {
		allCompilables.Compile()
	}, func() {
		// stop updating certificates
		allowedCerts.Stop()

		// close websockets first
		ws.Shutdown()

		// close http servers
		if srvApi != nil {
			_ = srvApi.Close()
		}
		if srvHttp != nil {
			_ = srvHttp.Close()
		}
		if srvHttps != nil {
			_ = srvHttps.Close()
		}
	})
}
