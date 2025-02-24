package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/evanw/esbuild/pkg/api"
	"segoqu.com/nova/internal/bundler"
	"segoqu.com/nova/internal/generator"
	"segoqu.com/nova/internal/project"
)

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

func dev() {
	command := func(ctx context.Context) *exec.Cmd {
		cmd := exec.CommandContext(ctx, "go", "run", project.Abs(filepath.Join(".nova", "main.go")))
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd
	}

	start := func(cmd *exec.Cmd) {
		err := cmd.Start()
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("[server] started http://localhost:8080")
	}

	stop := func(cmd *exec.Cmd) {
		cmd.Process.Signal(os.Interrupt)
		syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
		cmd.Wait()

		cmd.Process.Kill()
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		log.Println("[server] stoped")
	}

	generate := func(addr string) {
		routes := must(generator.FindRoutes("src/pages"))
		err := generator.GenerateServer(routes, generator.OptionProxy(addr))
		if err != nil {
			panic(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := command(ctx)

	esbuild := must(bundler.Development("src/pages"))
	defer esbuild.Dispose()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
		<-sig
		stop(cmd)
		esbuild.Dispose()
		cancel()
	}()

	err := esbuild.Watch(api.WatchOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	serve := must(esbuild.Serve(api.ServeOptions{Host: "127.0.0.1", Port: 0}))
	for _, host := range serve.Hosts {
		log.Println(
			"[esbuild]", "bundler server listening at",
			host+":"+strconv.Itoa(int(serve.Port)),
		)
	}

	addr := serve.Hosts[0] + ":" + strconv.Itoa(int(serve.Port))

	generate(addr)
	start(cmd)

	watcher := generator.NewWatcher(ctx, "*.go", func(s string) {
		log.Println("[reload]", s)

		stop(cmd)
		cmd = command(ctx)

		generate(addr)
		start(cmd)
	})
	go watcher.Watch("src/pages")

	<-ctx.Done()
}

func build() {
	esbuild := must(bundler.Production("src/pages"))
	defer esbuild.Dispose()

	result := esbuild.Rebuild()
	if len(result.Errors) > 0 {
		panic(result.Errors)
	}
	for _, file := range result.OutputFiles {
		log.Println(strings.TrimPrefix(file.Path, project.Root()+"/"))
	}

	routes := must(generator.FindRoutes("src/pages"))
	err := generator.GenerateServer(routes, generator.OptionStatic())
	if err != nil {
		panic(err)
	}

	log.Println("[build] go build -o .nova/app")
	in := filepath.Join(project.Root(), ".nova", "main.go")
	out := filepath.Join(project.Root(), ".nova", "app")
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

	case "build":
		build()

	default:
		help()
	}
}
