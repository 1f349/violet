package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/1f349/violet/domains"
	"github.com/1f349/violet/proxy"
	"github.com/1f349/violet/proxy/websocket"
	"github.com/1f349/violet/router"
	"github.com/1f349/violet/target"
	"github.com/AlecAivazis/survey/v2"
	"github.com/google/subcommands"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

type setupCmd struct {
	wdPath string
}

func (s *setupCmd) Name() string     { return "setup" }
func (s *setupCmd) Synopsis() string { return "Walkthrough creating a config file" }
func (s *setupCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&s.wdPath, "wd", ".", "Path to the directory to create config files in (defaults to the working directory)")
}
func (s *setupCmd) Usage() string {
	return `setup
  Setup Violet automatically by answering questions.
`
}

func (s *setupCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// get absolute path to specify files
	wdAbs, err := filepath.Abs(s.wdPath)
	if err != nil {
		fmt.Println("[Violet] Failed to get full directory path: ", err)
		return subcommands.ExitFailure
	}

	// ask about running the setup steps
	createFile := false
	err = survey.AskOne(&survey.Confirm{Message: fmt.Sprintf("Create Violet config files in this directory: '%s'?", wdAbs)}, &createFile)
	if err != nil {
		fmt.Println("[Violet] Error: ", err)
		return subcommands.ExitFailure
	}
	if !createFile {
		fmt.Println("[Violet] Goodbye")
		return subcommands.ExitSuccess
	}

	// store answers from questions
	var answers struct {
		SelfSigned  bool
		ErrorPages  bool
		ApiListen   string
		HttpListen  string
		HttpsListen string
		RateLimit   uint64
		FirstDomain string
		ApiUrl      string
	}

	// ask main questions
	err = survey.Ask([]*survey.Question{
		{
			Name:   "SelfSigned",
			Prompt: &survey.Confirm{Message: "Enable self-signed certificate?"},
		},
		{
			Name:   "ErrorPages",
			Prompt: &survey.Confirm{Message: "Enable custom error pages?"},
		},
		{
			Name:     "ApiListen",
			Prompt:   &survey.Input{Message: "API listen address", Default: "127.0.0.1:8080"},
			Validate: listenAddressValidator,
		},
		{
			Name:   "HttpListen",
			Prompt: &survey.Input{Message: "HTTP listen address", Default: ":80"},
		},
		{
			Name:   "HttpsListen",
			Prompt: &survey.Input{Message: "HTTPS listen address", Default: ":443"},
		},
		{
			Name:   "RateLimit",
			Prompt: &survey.Input{Message: "Rate limit", Default: "300", Help: "Number of allowed requests per minute per IP"},
			Validate: func(ans interface{}) error {
				if ansStr, ok := ans.(string); ok {
					_, err := strconv.ParseUint(ansStr, 10, 64)
					return err
				}
				return nil
			},
		},
		{
			Name:     "FirstDomain",
			Prompt:   &survey.Input{Message: "First domain", Default: "example.com", Help: "Setup the first domain or it will be more difficult to setup later"},
			Validate: survey.Required,
		},
	}, &answers)
	if err != nil {
		fmt.Println("[Violet] Error: ", err)
		return subcommands.ExitFailure
	}

	// generate database path
	databaseFile := filepath.Join(wdAbs, "violet.db.sqlite")
	errorPagePath := ""
	if answers.ErrorPages {
		errorPagePath = filepath.Join(wdAbs, "error-pages")
	}

	// write config file
	confFile := filepath.Join(wdAbs, "config.json")
	createConf, err := os.Create(confFile)
	if err != nil {
		return 0
	}
	confEncode := json.NewEncoder(createConf)
	confEncode.SetIndent("", "  ")
	err = confEncode.Encode(startUpConfig{
		SelfSigned:    answers.SelfSigned,
		ErrorPagePath: errorPagePath,
		Listen: listenConfig{
			Api:   answers.ApiListen,
			Http:  answers.HttpListen,
			Https: answers.HttpsListen,
		},
		InkscapeCmd: "inkscape",
		RateLimit:   answers.RateLimit,
	})
	if err != nil {
		fmt.Println("[Violet] Failed to write config file: ", err)
		return subcommands.ExitFailure
	}

	// open sqlite database
	db, err := sql.Open("sqlite3", databaseFile)
	if err != nil {
		log.Fatalf("[Violet] Failed to open database '%s'...", databaseFile)
	}

	// domain manager to add a domain, no need to compile here as the program needs
	// to be run again with the serve subcommand
	allowedDomains := domains.New(db)
	allowedDomains.Put(answers.FirstDomain, true)

	// don't bother with this part is the api won't be listening
	if answers.ApiListen != "" {
		// ask for url
		err = survey.AskOne(&survey.Input{Message: "API URL", Default: "api.example.com/violet", Help: "Enter the URL which should point to the internal Violet API"}, &answers.ApiUrl, survey.WithValidator(func(ans interface{}) error {
			if ansStr, ok := ans.(string); ok {
				_, err := url.Parse(ansStr)
				return err
			}
			return nil
		}))
		if err != nil {
			fmt.Println("[Violet] Error: ", err)
			return subcommands.ExitFailure
		}

		// parse the api url
		apiUrl, err := url.Parse(answers.ApiUrl)
		if err != nil {
			fmt.Println("[Violet] Failed to parse API URL: ", err)
			return subcommands.ExitFailure
		}

		// add with the route manager, no need to compile as this will run when opened
		// with the serve subcommand
		routeManager := router.NewManager(db, proxy.NewHybridTransportWithCalls(&nilTransport{}, &nilTransport{}, &websocket.Server{}))
		err = routeManager.InsertRoute(target.RouteWithActive{
			Route: target.Route{
				Src:   path.Join(apiUrl.Host, apiUrl.Path),
				Dst:   answers.ApiListen,
				Flags: target.FlagPre | target.FlagCors | target.FlagForwardHost | target.FlagForwardAddr,
			},
			Active: true,
		})
		if err != nil {
			fmt.Println("[Violet] Failed to insert api route into database: ", err)
			return subcommands.ExitFailure
		}
	}

	fmt.Println("[Violet] Setup complete")
	fmt.Printf("[Violet] Run the reverse proxy with `violet serve -conf %s`\n", confFile)

	return subcommands.ExitSuccess
}

func listenAddressValidator(ans interface{}) error {
	if ansStr, ok := ans.(string); ok {
		// empty string means disable
		if ansStr == "" {
			return nil
		}

		// use ResolveTCPAddr to validate the input
		_, err := net.ResolveTCPAddr("tcp", ansStr)
		return err
	}
	return nil
}

type nilTransport struct{}

func (n *nilTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("not sure how you are sending a request")
}
