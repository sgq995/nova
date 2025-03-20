package project

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"slices"

	"github.com/sgq995/nova/internal/codegen"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/esbuild"
	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/router"
	"github.com/sgq995/nova/internal/watcher"
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
	e := esbuild.NewESBuild(p.config)
	r := router.NewRouter(p.config)
	c := codegen.NewCodegen(p.config)
	runner := newRunner(p.config)

	logger.Infof("starting nova dev server...")
	// if err := rebuild(scanner, r, c); err != nil {
	// 	return nil, err
	// }
	scanner.scan()

	static := slices.Concat(scanner.jsFiles, scanner.cssFiles)
	esbuildCtx := e.Context(static)
	server := &serverImpl{
		scanner: scanner,
		router:  r,
		codegen: c,
		runner:  runner,
		esbuild: esbuildCtx,
	}

	staticFiles, err := esbuildCtx.Build()
	if err != nil {
		return nil, err
	}

	files := map[string][]byte{"@nova/hmr.js": hmr}
	root := module.Join(p.config.Codegen.OutDir, "static")
	for filename, contents := range staticFiles {
		name, _ := filepath.Rel(root, filename)
		files[name] = contents
	}

	if err := runner.start(files); err != nil {
		logger.Errorf("%+v", err)
	}

	go watcher.WatchDir(ctx, p.config.Router.Pages, watcher.CallbackMap{
		"*.go": {
			OnCreate: func(filename string) error {
				logger.Infof("create (%s)", filename)

				routes, err := server.router.ParseRoute(filename)
				if err != nil {
					return err
				}

				err = server.codegen.GenerateModule(filename, routes)
				if err != nil {
					return err
				}

				for _, route := range routes {
					switch route := route.(type) {
					case *router.RenderRouteGo:
						runner.send(codegen.CreateRouteMessage(route.Pattern))

					case *router.RestRouteGo:
						runner.send(codegen.CreateRouteMessage(route.Pattern))
					}
				}

				return nil
			},
			OnUpdate: func(filename string) error {
				logger.Infof("update (%s)", filename)

				routes, err := server.router.ParseRoute(filename)
				if err != nil {
					return err
				}

				err = server.codegen.GenerateModule(filename, routes)
				if err != nil {
					return err
				}

				for _, route := range routes {
					switch route := route.(type) {
					case *router.RenderRouteGo:
						runner.send(codegen.CreateRouteMessage(route.Pattern))

					case *router.RestRouteGo:
						runner.send(codegen.CreateRouteMessage(route.Pattern))
					}
				}

				runner.send(codegen.UpdateFileMessage(filename, []byte{}))

				return nil
			},
			OnDelete: func(filename string) error {
				routes := server.router.Routes[filename]
				for _, route := range routes {
					switch route := route.(type) {
					case *router.RenderRouteGo:
						runner.send(codegen.DeleteRouteMessage(route.Pattern))

					case *router.RestRouteGo:
						runner.send(codegen.DeleteRouteMessage(route.Pattern))
					}
				}
				return nil
			},
		},
		"*.js,*.ts,*.css": {
			OnUpdate: func(filename string) error {
				// TODO: buffer for files added/modified in error state
				logger.Infof("change (%s)", filename)

				root := module.Abs(p.config.Router.Pages)
				in, _ := filepath.Rel(root, filename)
				out := module.Join(p.config.Codegen.OutDir, "static", in)
				files, err := server.esbuild.Build()
				if err != nil {
					logger.Errorf("%+v", err)
					return err
				}
				if contents, exists := files[out]; exists {
					runner.send(codegen.UpdateFileMessage(in, contents))
				} else {
					server.esbuild.Dispose()

					err := scanner.scan()
					if err != nil {
						logger.Errorf("%+v", err)
						return err
					}

					static := slices.Concat(scanner.jsFiles, scanner.cssFiles)
					server.esbuild = e.Context(static)
					files, err := server.esbuild.Build()
					if err != nil {
						logger.Errorf("%+v", err)
						return err
					}
					runner.send(codegen.UpdateFileMessage(in, files[out]))
				}

				return nil
			},
		},
		"*.html": {
			OnUpdate: func(filename string) error {
				logger.Infof("change (%s)", filename)
				runner.send(codegen.UpdateFileMessage(filename, []byte{}))
				return nil
			},
		},
	})

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
	staticDir := module.Join(p.config.Codegen.OutDir, "static")
	staticEntryMap, err := e.Build(esbuild.BuildOptions{
		EntryPoints: static,
		Outdir:      staticDir,
		Hashing:     true,
	})
	if err != nil {
		return err
	}

	templates := s.templateFiles
	templatesDir := module.Join(p.config.Codegen.OutDir, "templates")
	_, err = e.Build(esbuild.BuildOptions{
		EntryPoints: templates,
		Outdir:      templatesDir,
		EntryMap:    staticEntryMap,
	})
	if err != nil {
		return err
	}

	pages := []string{}
	for _, page := range s.htmlFiles {
		if !slices.Contains(s.templateFiles, page) {
			pages = append(pages, page)
		}
	}
	pagesDir := module.Join(p.config.Codegen.OutDir, "pages")
	_, err = e.Build(esbuild.BuildOptions{
		EntryPoints: pages,
		Outdir:      pagesDir,
		EntryMap:    staticEntryMap,
	})
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
