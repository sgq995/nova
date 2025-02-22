package main

import (
	"context"
	"fmt"
	"os"

	"segoqu.com/nova/internal/generator"
)

func main() {
	fmt.Println(os.Args)

	routes, _ := generator.FindRoutes("src/pages")
	fmt.Println(routes)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := generator.NewWatcher(ctx, "*.go", func(s string) {
		fmt.Println(s)
	})
	go watcher.Watch("src/pages")

	<-ctx.Done()
}
