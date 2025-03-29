package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/fsys"
	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/must"
	"github.com/sgq995/nova/internal/project"
)

func dev(c config.Config) {
	nova := must.Must(project.Context(c))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := must.Must(nova.Serve(ctx))
	defer server.Dispose()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM)

	go func() {
		<-sig
		server.Dispose()
		cancel()
	}()

	<-ctx.Done()
}

func build(c config.Config) {
	nova := must.Must(project.Context(c))
	err := nova.Build()
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}

	// TODO: move go build execution to nova.Build
	in := module.Join(c.Codegen.OutDir, "main.go")
	out := module.Join(c.Codegen.OutDir, "app")
	cmd := exec.Command("go", "build", "-o", out, in)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Infof("go build -o %s %s", out, in)
	err = cmd.Run()
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}

	logger.Infof("success (%s)", out)
}

func initCmd() {
	filename := module.Abs("nova.config.json")

	if must.Must(fsys.FileExists(filename)) {
		logger.Errorf("nova.config.json already exists")
		return
	}

	file := must.Must(os.Create(filename))
	defer file.Close()

	cfg := config.Default()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(cfg)

	logger.Infof("nova.config.json created")
}

func help() {
	flag.Usage()
	fmt.Fprintf(flag.CommandLine.Output(), "\n%s %s\n", os.Args[0], "dev|build")
}

func main() {
	configFile := flag.String(
		"config-file",
		module.Abs("nova.config.json"),
		"A JSON config file",
	)

	flag.Parse()

	cfg := config.Default()
	if *configFile != "" {
		other, err := config.Read(*configFile)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
		cfg.Merge(&other)
	}

	args := flag.Args()
	if len(args) < 1 {
		help()
		return
	}

	switch args[0] {
	case "dev":
		dev(cfg)

	case "build":
		build(cfg)

	case "init":
		initCmd()

	default:
		help()
	}
}
