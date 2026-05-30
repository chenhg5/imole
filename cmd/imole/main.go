package main

import (
	"context"
	"os"
	"time"

	"github.com/chenhg5/imole/internal/cli"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	app := cli.New(os.Stdout, os.Stderr)
	os.Exit(app.Run(ctx, os.Args[1:]))
}