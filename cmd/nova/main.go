package main

import (
	"context"
	"fmt"
	"os"

	"github.com/evanw/esbuild/pkg/api"
	"segoqu.com/nova/internal/bundler"
	"segoqu.com/nova/internal/generator"
)

func main() {
	fmt.Println(os.Args)

	routes, _ := generator.FindRoutes("src/pages")
	fmt.Println(routes)

	generator.GenerateServer(routes)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := generator.NewWatcher(ctx, "*.go", func(s string) {
		fmt.Println(s)
	})
	go watcher.Watch("src/pages")

	esbuild, _ := bundler.Development("src/pages")
	serve, _ := esbuild.Serve(api.ServeOptions{Port: 0})
	fmt.Println(serve)
	defer esbuild.Dispose()

	<-ctx.Done()
}
