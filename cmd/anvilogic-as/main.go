// Command anvilogic-as is a minimalist, agent-first CLI for the Anvilogic
// SaaS control-plane API, driven by the embedded api-surface registry.
package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/grandcamel/anvilogic-as/internal/cli"
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	v := version
	if v == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if mv := bi.Main.Version; mv != "" && mv != "(devel)" {
				v = mv
			}
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(cli.Execute(ctx, v))
}
