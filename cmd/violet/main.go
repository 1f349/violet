package main

import (
	"context"
	_ "embed"
	"flag"
	"github.com/google/subcommands"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&serveCmd{}, "")
	subcommands.Register(&setupCmd{}, "")
	subcommands.Register(&tokenCmd{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
