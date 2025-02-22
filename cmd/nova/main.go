package main

import (
	"fmt"
	"os"

	"segoqu.com/nova/internal/generator"
)

func main() {
	fmt.Println(os.Args)

	routes, _ := generator.FindRoutes("src/pages")
	fmt.Println(routes)
}
