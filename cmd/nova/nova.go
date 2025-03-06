package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sgq995/nova/internal/bundler"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/generator"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/project"
	"github.com/sgq995/nova/internal/utils"
)

func dev(c config.Config) {
	nova, _ := project.Context(c)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := nova.Serve(ctx)
	if err != nil {
		log.Fatalln("fatal", err)
	}
	defer server.Dispose()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
		<-sig
		server.Dispose()
		cancel()
	}()

	log.Printf("[nova] http://%s:%d\n", c.Server.Host, c.Server.Port)
	<-ctx.Done()
}

func build() {
	esbuild := utils.Must(bundler.Production("src/pages"))
	defer esbuild.Dispose()

	result := esbuild.Rebuild()
	if len(result.Errors) > 0 {
		errs := []error{}
		for _, e := range result.Errors {
			errs = append(errs, errors.New(e.Text))
		}
		panic(errors.Join(errs...))
	}
	for _, file := range result.OutputFiles {
		log.Println(strings.TrimPrefix(file.Path, module.Root()+"/"))
	}

	routes := utils.Must(generator.FindRoutes("src/pages"))
	err := generator.GenerateServer(routes, generator.OptionStatic(), generator.OptionEnvironment(generator.Prod))
	if err != nil {
		panic(err)
	}

	log.Println("[build] go build -o .nova/app")
	in := filepath.Join(module.Root(), ".nova", "main.go")
	out := filepath.Join(module.Root(), ".nova", "app")
	cmd := exec.Command("go", "build", "-o", out, in)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("[build] done")
}

func help() {
	flag.Usage()
	fmt.Fprintf(flag.CommandLine.Output(), "\n%s %s\n", os.Args[0], "dev|build")
}

func main() {
	configFile := flag.String(
		"config-file",
		filepath.Join(module.Root(), "nova.config.json"),
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
		build()

	default:
		help()
	}
}
