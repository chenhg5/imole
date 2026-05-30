package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chenhg5/imole/internal/cli"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	app := cli.New(os.Stdout, os.Stderr)
	if err := app.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "imole: %v\n", err)
		os.Exit(1)
	}
}
