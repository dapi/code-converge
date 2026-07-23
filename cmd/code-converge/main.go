package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/dapi/code-converge/internal/app"
	"github.com/dapi/code-converge/internal/repository"
)

func main() {
	if repository.IsScopedGitWrapperInvocation(os.Args[0]) {
		os.Exit(repository.RunScopedGitWrapper(os.Args[1:]))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(app.App{}.Run(ctx, os.Args[1:]))
}
