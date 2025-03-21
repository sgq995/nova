package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/project"
	"github.com/sgq995/nova/internal/utils"
)

func dev(c config.Config) {
	nova := utils.Must(project.Context(c))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := utils.Must(nova.Serve(ctx))
	defer server.Dispose()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
		<-sig
		server.Dispose()
		cancel()
	}()

	<-ctx.Done()
}

func build(c config.Config) {
	nova := utils.Must(project.Context(c))
	err := nova.Build()
	if err != nil {
		logger.Errorf("%+v", err)
		return
	}

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

	c := config.Default()
	if *configFile != "" {
		c2, err := config.Read(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		c = c2
	}

	args := flag.Args()
	if len(args) < 1 {
		help()
		return
	}

	switch args[0] {
	case "dev":
		dev(c)

	case "build":
		build(c)

	default:
		help()
	}
}
