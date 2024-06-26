package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/1f349/mjwt"
	"github.com/1f349/violet"
	"github.com/1f349/violet/certs"
	"github.com/1f349/violet/domains"
	errorPages "github.com/1f349/violet/error-pages"
	"github.com/1f349/violet/favicons"
	"github.com/1f349/violet/logger"
	"github.com/1f349/violet/proxy"
	"github.com/1f349/violet/proxy/websocket"
	"github.com/1f349/violet/router"
	"github.com/1f349/violet/servers"
	"github.com/1f349/violet/servers/api"
	"github.com/1f349/violet/servers/conf"
	"github.com/1f349/violet/utils"
	"github.com/google/subcommands"
	"github.com/mrmelon54/exit-reload"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

type serveCmd struct {
	configPath string
	cpuprofile string
}

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

func (s *serveCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	logger.Logger.Info("Starting...")

	if s.configPath == "" {
		logger.Logger.Info("Error: config flag is missing")
		return subcommands.ExitUsageError
	}

	openConf, err := os.Open(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Logger.Info("Error: missing config file")
		} else {
			logger.Logger.Info("Error: open config file: ", err)
		}
		return subcommands.ExitFailure
	}

	var config startUpConfig
	err = json.NewDecoder(openConf).Decode(&config)
	if err != nil {
		logger.Logger.Info("Error: invalid config file: ", err)
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
			logger.Logger.Fatal("Failed to create certificate path")
		}
		// create path to key dir
		err = os.MkdirAll(filepath.Join(wd, "keys"), os.ModePerm)
		if err != nil {
			logger.Logger.Fatal("Failed to create certificate key path")
		}
	}

	// errorPageDir stores an FS interface for accessing the error page directory
	var errorPageDir fs.FS
	if startUp.ErrorPagePath != "" {
		errorPageDir = os.DirFS(startUp.ErrorPagePath)
		err := os.MkdirAll(startUp.ErrorPagePath, os.ModePerm)
		if err != nil {
			logger.Logger.Fatal("Failed to create error page", "path", startUp.ErrorPagePath)
		}
	}

	// load the MJWT RSA public key from a pem encoded file
	mJwtVerify, err := mjwt.NewMJwtVerifierFromFile(filepath.Join(wd, "signer.public.pem"))
	if err != nil {
		logger.Logger.Fatal("Failed to load MJWT verifier public key", "file", filepath.Join(wd, "signer.public.pem"), "err", err)
	}

	// open sqlite database
	db, err := violet.InitDB(filepath.Join(wd, "violet.db.sqlite"))
	if err != nil {
		logger.Logger.Fatal("Failed to open database", "err", err)
	}

	certDir := os.DirFS(filepath.Join(wd, "certs"))
	keyDir := os.DirFS(filepath.Join(wd, "keys"))

	// setup registry for metrics
	promRegistry := prometheus.NewRegistry()
	promRegistry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

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
		srvApi = api.NewApiServer(srvConf, allCompilables, promRegistry)
		srvApi.SetKeepAlivesEnabled(false)
		l := logger.Logger.With("server", "API")
		l.Info("Starting server", "addr", srvApi.Addr)
		go utils.RunBackgroundHttp(l, srvApi)
	}
	if srvConf.HttpListen != "" {
		srvHttp = servers.NewHttpServer(srvConf, promRegistry)
		srvHttp.SetKeepAlivesEnabled(false)
		l := logger.Logger.With("server", "HTTP")
		l.Info("Starting server", "addr", srvHttp.Addr)
		go utils.RunBackgroundHttp(l, srvHttp)
	}
	if srvConf.HttpsListen != "" {
		srvHttps = servers.NewHttpsServer(srvConf, promRegistry)
		srvHttps.SetKeepAlivesEnabled(false)
		l := logger.Logger.With("server", "HTTPS")
		l.Info("Starting server", "addr", srvHttps.Addr)
		go utils.RunBackgroundHttps(l, srvHttps)
	}

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
