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
var hmr []byte

type Server interface {
	Dispose()
}

type ProjectContext interface {
	Serve(ctx context.Context) (Server, error)

	Build() error
}

func Context(c config.Config) (ProjectContext, error) {
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

	staticFiles, err := esbuildCtx.Build()
	if err != nil {
		return nil, err
	}

	files := map[string][]byte{"@nova/hmr.js": hmr}
	root := module.Abs(filepath.Join(p.config.Codegen.OutDir, "static"))
	for filename, contents := range staticFiles {
		name, _ := filepath.Rel(root, filename)
		files[name] = contents
	}

	runner.start(files)

	watcher := newWatcher(ctx, map[string]func(string){
		"*.go": func(filename string) {
			rebuild(server.scanner, server.router, server.codegen)

			files := map[string][]byte{
				filename:       {},
				"@nova/hmr.js": hmr,
			}
			root := module.Abs(filepath.Join(p.config.Codegen.OutDir, "static"))
			for filename, contents := range staticFiles {
				name, _ := filepath.Rel(root, filename)
				files[name] = contents
			}

			runner.restart(files)

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
			runner.update(name, []byte{})
		},
	})
	go watcher.watch(p.config.Router.Pages)

	return server, nil
}

func (p *projectContextImpl) Build() error {
	os.Setenv("NOVA_ENV", "production")

	e := esbuild.NewESBuild(p.config)
	s := newScanner(p.config)
	r := router.NewRouter(p.config)
	c := codegen.NewCodegen(p.config)

	if err := s.scan(); err != nil {
		return err
	}

	static := slices.Concat(s.jsFiles, s.cssFiles)
	staticDir := module.Abs(filepath.Join(p.config.Codegen.OutDir, "static"))
	staticEntryMap, err := e.Build(esbuild.BuildOptions{
		EntryPoints: static,
		Outdir:      staticDir,
		Hashing:     true,
	})
	if err != nil {
		return err
	}

	templates := s.templateFiles
	templatesDir := module.Abs(filepath.Join(p.config.Codegen.OutDir, "templates"))
	_, err = e.Build(esbuild.BuildOptions{
		EntryPoints: templates,
		Outdir:      templatesDir,
		EntryMap:    staticEntryMap,
	})

	pages := []string{}
	for _, page := range s.htmlFiles {
		if !slices.Contains(s.templateFiles, page) {
			pages = append(pages, page)
		}
	}
	pagesDir := module.Abs(filepath.Join(p.config.Codegen.OutDir, "pages"))
	_, err = e.Build(esbuild.BuildOptions{
		EntryPoints: pages,
		Outdir:      pagesDir,
		EntryMap:    staticEntryMap,
	})

	if err := rebuild(s, r, c); err != nil {
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
