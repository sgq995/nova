package project

import (
	"context"
	_ "embed"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/sgq995/nova/internal/codegen"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/esbuild"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/router"
)

//go:embed hmr.js
var hmr string

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

	rebuild(scanner, router, codegen)

	static := slices.Concat(scanner.jsFiles, scanner.cssFiles)
	esbuildCtx := esbuild.Context(static)
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

	runner.start()

	runner.update("@nova/hmr.js", hmr)
	root := module.Abs(filepath.Join(p.config.Codegen.OutDir, "static"))
	for filename, contents := range files {
		name, _ := filepath.Rel(root, filename)
		runner.update(name, contents)
	}

	watcher := newWatcher(ctx, map[string]func(string){
		"*.go": func(filename string) {
			rebuild(server.scanner, server.router, server.codegen)

			runner.restart()

			runner.update(filename, "")
			runner.update("@nova/hmr.js", hmr)
			root := module.Abs(filepath.Join(p.config.Codegen.OutDir, "static"))
			for filename, contents := range files {
				name, _ := filepath.Rel(root, filename)
				runner.update(name, contents)
			}

			log.Println("[reload]", filename)
		},
		"*.js,*.jsx,*.ts,*.tsx,*.css": func(name string) {
			root := module.Abs(p.config.Router.Pages)
			in, _ := filepath.Rel(root, name)
			out := module.Abs(filepath.Join(p.config.Codegen.OutDir, "static", in))
			files, err := server.esbuild.Build()
			if err != nil {
				log.Println("[esbuild]", err)
				return
			}
			if contents, exists := files[out]; exists {
				log.Println("[reload]", name)
				runner.update(in, contents)
			} else {
				server.esbuild.Dispose()

				err := scanner.scan()
				if err != nil {
					log.Println("[scanner]", err)
					return
				}

				static := slices.Concat(scanner.jsFiles, scanner.cssFiles)
				server.esbuild = esbuild.Context(static)
				files, err := server.esbuild.Build()
				if err != nil {
					log.Println("[esbuild]", err)
					return
				}
				runner.update(in, files[out])
			}
		},
		"*.html": func(name string) {
			log.Println("[reload]", name)
			runner.update(name, "")
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

func (p *projectContextImpl) Build() error {
	os.Setenv("NOVA_ENV", "production")

	scanner := newScanner(p.config)
	// esbuild := esbuild.NewESBuild(p.config)
	router := router.NewRouter(p.config)
	codegen := codegen.NewCodegen(p.config)

	err := rebuild(scanner, router, codegen)
	if err != nil {
		return err
	}

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

func rebuild(s *scanner, r *router.Router, c *codegen.Codegen) error {
	err := s.scan()
	if err != nil {
		return err
	}
	routes, err := r.ParseRoutes(s.pages)
	if err != nil {
		return err
	}
	err = c.Generate(routes)
	if err != nil {
		return err
	}
	return nil
}
