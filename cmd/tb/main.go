package main

import "github.com/pmqueiroz/toolbelt/internal/cli"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.Execute(cli.Build{Version: version, Commit: commit, Date: date})
}
