package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/MrMelon54/mjwt"
	"github.com/google/subcommands"
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

type tokenCmd struct {
	configPath string
	audience   stringSliceFlag
	duration   string
	permission stringSliceFlag
}

func (t *tokenCmd) Name() string { return "auth" }
func (t *tokenCmd) Synopsis() string {
	return "Generate an auth token for using the API"
}
func (t *tokenCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&t.configPath, "conf", "", "/path/to/config.json : path to the config file")
	f.Var(&t.audience, "a", "specify the audience attribute, this flag can be used multiple times")
	f.StringVar(&t.duration, "d", "15m", "specify the duration attribute (default: 15m)")
	f.Var(&t.permission, "p", "specify the permissions granted by this token, this flag can be used multiple times")
}
func (t *tokenCmd) Usage() string {
	return `token [-conf <config file>] [-a <audience>] [-d <duration>] [-p <permission>]
  Generate an access/refresh token pair for using the API.
`
}

func (t *tokenCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	err := t.normal()
	if err != nil {
		fmt.Println("[Violet] Error: ", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

func (t *tokenCmd) normal() error {
	wd := filepath.Dir(t.configPath)
	openConfig, err := os.Open(t.configPath)
	if err != nil {
		return err
	}

	var a struct {
		SignerIssuer string `json:"signer_issuer"`
	}
	err = json.NewDecoder(openConfig).Decode(&a)
	if err != nil {
		return err
	}

	signer, err := mjwt.NewMJwtSignerFromFile(a.SignerIssuer, filepath.Join(wd, "violet.private.pem"))
	if err != nil {
		return err
	}

	signer.GenerateJwt(uuid.NewString())
}

type stringSliceFlag []string

func (a *stringSliceFlag) String() string {
	return fmt.Sprintf("%v", *a)
}
func (a *stringSliceFlag) Set(s string) error {
	*a = append(*a, s)
	return nil
}
