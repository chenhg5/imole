package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
)

const Version = "0.1.0"

type App struct {
	out io.Writer
	err io.Writer
}

func New(out, err io.Writer) *App {
	return &App{out: out, err: err}
}

func (a *App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return a.runHelp()
	}

	switch args[0] {
	case "doctor":
		return a.runDoctor(ctx, args[1:])
	case "scan":
		return a.runScan(ctx, args[1:])
	case "videos":
		return a.runVideos(ctx, args[1:])
	case "backup":
		return a.runBackup(ctx, args[1:])
	case "report":
		return a.runReport(ctx, args[1:])
	case "guide":
		return a.runGuide(ctx, args[1:])
	case "clean":
		return a.runClean(ctx, args[1:])
	case "help", "--help", "-h":
		return a.runHelp()
	case "version", "--version", "-v":
		fmt.Fprintf(a.out, "imole %s\n", Version)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\nRun: imole help", args[0])
	}
}

func (a *App) runHelp() error {
	_, err := fmt.Fprint(a.out, `iMole - open-source iPhone slimming toolkit

Usage:
  imole doctor                         Check device and local dependencies
  imole scan [provider flags] [--json] Scan visible iPhone media
  imole videos [--top N]               Show largest videos
  imole backup --to PATH [filters]     Back up media and write manifest
  imole report --manifest PATH         Summarize a backup manifest
  imole guide [topic]                  Show cleanup guidance
  imole clean                          Explain safe cleanup boundaries

Common filters:
  --provider auto|filesystem|imagecapture|gphoto
  --source PATH
  --only all|photos|videos
  --older-than 90d
  --large-than 500MB

Notes:
  iMole v0.1 focuses on diagnosis, backup, verification, and guidance.
  It does not automatically delete iPhone Photos library content by default.
`)
	return err
}

func usageError(msg string) error {
	if msg == "" {
		return errors.New("invalid usage")
	}
	return errors.New(msg)
}
