package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/evanw/esbuild/pkg/api"
	"segoqu.com/nova/internal/bundler"
	"segoqu.com/nova/internal/generator"
)

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

func dev() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	esbuild := must(bundler.Development("src/pages"))
	defer esbuild.Dispose()

	generate := func(addr string) {
		routes := must(generator.FindRoutes("src/pages"))
		err := generator.GenerateServer(routes, generator.OptionProxy(addr))
		if err != nil {
			panic(err)
		}
	}

	serve := must(esbuild.Serve(api.ServeOptions{Port: 0}))
	for _, host := range serve.Hosts {
		log.Println(
			"[esbuild]", "bundler server listening at",
			host+":"+strconv.Itoa(int(serve.Port)),
		)
	}

	addr := serve.Hosts[0] + ":" + strconv.Itoa(int(serve.Port))

	generate(addr)
	watcher := generator.NewWatcher(ctx, "*.go", func(s string) {
		log.Println("[reload]", s)
		generate(addr)
	})
	go watcher.Watch("src/pages")

	<-ctx.Done()
}

func help() {
	flag.Usage()
	fmt.Fprintf(flag.CommandLine.Output(), "\t%s %s\n", os.Args[0], "dev|build")
}

func main() {
	// configFileSet := flag.NewFlagSet("config-file", flag.ContinueOnError)
	// configFile := configFileSet.String(
	// 	"config-file",
	// 	filepath.Join(project.Root(), "nova.config.json"),
	// 	"A JSON config file",
	// )
	// configFileSet.Parse(os.Args)

	devCmd := flag.NewFlagSet("dev", flag.ExitOnError)

	flag.Parse()

	if len(os.Args) < 2 {
		help()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "dev":
		devCmd.Parse(os.Args[2:])
		dev()

	default:
		help()
	}
}
