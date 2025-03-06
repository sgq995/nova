package project

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sgq995/nova/internal/codegen"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/esbuild"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/router"
)

type Server interface {
	Dispose()
}

type ProjectContext interface {
	Serve(ctx context.Context) (Server, error)

	Build() error
}

func Context(c config.Config) (ProjectContext, error) {
	// Scan should Link as those are correlated
	// files := Scan(pages)
	// Link(files)

	// CreateRouter and Exec should be merged into Generate
	// router := CreateRouter(files, WithProxy(proxyUrl))
	// Execute(mainTemplate, router)

	return &projectContextImpl{config: &c}, nil
}

type serverImpl struct {
	scanner *scanner
	router  *router.Router
	codegen *codegen.Codegen
	runner  *runner
	esbuild esbuild.ESBuildContext
}

func (s *serverImpl) Dispose() {
	s.esbuild.Dispose()
	s.runner.stop()
}

type projectContextImpl struct {
	config *config.Config
}

func (p *projectContextImpl) Serve(ctx context.Context) (Server, error) {
	os.Setenv("NOVA_ENV", "development")

	scanner := newScanner(p.config)
	esbuild := esbuild.NewESBuild(p.config)
	router := router.NewRouter(p.config)
	codegen := codegen.NewCodegen(p.config)
	runner := newRunner(p.config)

	err := scanner.scan()
	if err != nil {
		return nil, err
	}
	routes, err := router.ParseRoutes(scanner.pages)
	if err != nil {
		return nil, err
	}
	err = codegen.Generate(routes)
	if err != nil {
		return nil, err
	}

	runner.create()
	runner.start()

	esbuildCtx := esbuild.Context(scanner.jsFiles)
	server := &serverImpl{
		scanner: scanner,
		router:  router,
		codegen: codegen,
		runner:  runner,
		esbuild: esbuildCtx,
	}

	files, err := esbuildCtx.Build()
	if err != nil {
		return nil, err
	}

	root := module.Abs(filepath.Join(p.config.Codegen.OutDir, "static"))
	for filename, contents := range files {
		name, _ := filepath.Rel(root, filename)
		runner.update(name, contents)
	}

	watcher := newWatcher(ctx, map[string]func(string){
		"*.go": func(filename string) {
			runner.stop()

			err := scanner.scan()
			if err != nil {
				log.Println("[scanner]", err)
				return
			}
			routes, err := router.ParseRoutes(scanner.pages)
			if err != nil {
				log.Println("[router]", err)
				return
			}
			err = codegen.Generate(routes)
			if err != nil {
				log.Println("[codegen]", err)
				return
			}
			log.Println("[reload]", filename)

			runner.create()
			runner.start()
		},
		"*.js": func(name string) {
			root := module.Abs(p.config.Router.Pages)
			name, _ = filepath.Rel(root, name)
			filename := module.Abs(filepath.Join(p.config.Codegen.OutDir, "static", name))
			files, err := server.esbuild.Build()
			if err != nil {
				log.Println("[esbuild]", err)
				return
			}
			fmt.Println("filename", filename)
			runner.update(name, files[filename])
		},
	})

	go watcher.watch(p.config.Router.Pages)

	// proxyUrl := BundlerServe(files)
	// Watch()
	// -> files := Scan(pages)
	// -> Link(files)
	// -> Bundle(files)
	// -> router := CreateRouter(files, WithProxy(proxyUrl))
	// -> Execute(mainTemplate, router)
	// -> Serve()

	return server, nil
}

func (*projectContextImpl) Build() error {
	os.Setenv("NOVA_ENV", "production")

	// build
	// files := Scan(c.Pages)
	// link with file handlers,
	// html will search for script and link,
	// js and css are automatically link by esbuild
	// assets will only be referenced
	// Link(files)
	// Bundle()
	// CreateRouter()
	// Execute(mainTemplate)
	// Build()

	return nil
}
