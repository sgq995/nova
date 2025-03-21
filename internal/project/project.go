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
	esbuild *esbuild.ESBuildContext
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
	e := esbuild.NewESBuildContext(p.config)
	r := router.NewRouter(p.config)
	c := codegen.NewCodegen(p.config)
	runner := newRunner(p.config)

	logger.Infof("starting nova dev server...")
	// if err := rebuild(scanner, r, c); err != nil {
	// 	return nil, err
	// }
	scanner.scan()

	static := slices.Concat(scanner.jsFiles, scanner.cssFiles)
	if err := e.Define(static); err != nil {
		return nil, err
	}

	server := &serverImpl{
		scanner: scanner,
		router:  r,
		codegen: c,
		runner:  runner,
		esbuild: e,
	}

	// staticFiles, err := esbuildCtx.Build()
	// if err != nil {
	// 	return nil, err
	// }

	files := map[string][]byte{"@nova/hmr.js": hmr}
	// root := module.Join(p.config.Codegen.OutDir, "static")
	// for filename, contents := range staticFiles {
	// 	name, _ := filepath.Rel(root, filename)
	// 	files[name] = contents
	// }

	if err := c.GenerateDevelopmentServer(); err != nil {
		logger.Errorf("%+v", err)
	}

	if err := runner.start(files); err != nil {
		logger.Errorf("%+v", err)
	}

	// TODO: make sure modifications are thread safe
	go watcher.WatchDir(ctx, p.config.Router.Pages, watcher.WatchCallbackMap{
		"*.go": func(event watcher.WatchEvent, files []string) error {
			// BUG: intentional filename declaration just to work in meantime
			filename := files[0]

			switch event {
			case watcher.CreateEvent:
				logger.Infof("create (%d)", len(files))

				routes, err := server.router.ParseRoute(filename)
				if err != nil {
					return err
				}

				err = server.codegen.GenerateRouteModule(filename, routes)
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

			case watcher.UpdateEvent:
				logger.Infof("update (%d)", len(files))

				routes, err := server.router.ParseRoute(filename)
				if err != nil {
					return err
				}

				err = server.codegen.GenerateRouteModule(filename, routes)
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

			case watcher.DeleteEvent:
				logger.Infof("delete (%s)", filename)

				routes := server.router.Routes[filename]
				for _, route := range routes {
					switch route := route.(type) {
					case *router.RenderRouteGo:
						runner.send(codegen.DeleteRouteMessage(route.Pattern))

					case *router.RestRouteGo:
						runner.send(codegen.DeleteRouteMessage(route.Pattern))
					}
				}
			}

			return nil
		},
		"*.js,*.ts,*.css": func(event watcher.WatchEvent, files []string) error {
			// BUG: intentional filename declaration just to work in meantime
			filename := files[0]

			switch event {
			case watcher.CreateEvent:
				logger.Infof("create (%s)", filename)

				err := scanner.scan()
				if err != nil {
					return err
				}

				// TODO: make thread safe esbuild wrapper for context
				server.esbuild.Dispose()
				static := slices.Concat(scanner.jsFiles, scanner.cssFiles)
				err = server.esbuild.Define(static)
				if err != nil {
					return err
				}

				files, err := server.esbuild.Build()
				if err != nil {
					return err
				}

				root := module.Abs(p.config.Router.Pages)
				in, err := filepath.Rel(root, filename)
				if err != nil {
					return err
				}
				out := module.Join(p.config.Codegen.OutDir, "static", in)

				contents := files[out]
				runner.send(codegen.UpdateFileMessage(in, contents))

			case watcher.UpdateEvent:
				logger.Infof("update (%s)", filename)

				files, err := server.esbuild.Build()
				if err != nil {
					return err
				}

				root := module.Abs(p.config.Router.Pages)
				in, err := filepath.Rel(root, filename)
				if err != nil {
					return err
				}
				out := module.Join(p.config.Codegen.OutDir, "static", in)

				contents := files[out]
				runner.send(codegen.UpdateFileMessage(in, contents))

			case watcher.DeleteEvent:
				logger.Infof("delete (%s)", filename)

				err := scanner.scan()
				if err != nil {
					return err
				}

				// TODO: make thread safe esbuild wrapper for context
				server.esbuild.Dispose()
				static := slices.Concat(scanner.jsFiles, scanner.cssFiles)
				err = server.esbuild.Define(static)
				if err != nil {
					return err
				}

				_, err = server.esbuild.Build()
				if err != nil {
					return err
				}

				root := module.Abs(p.config.Router.Pages)
				in, err := filepath.Rel(root, filename)
				if err != nil {
					return err
				}

				runner.send(codegen.DeleteFileMessage(in))
			}

			return nil
		},
		"*.html": func(event watcher.WatchEvent, files []string) error {
			// BUG: intentional filename declaration just to work in meantime
			filename := files[0]

			switch event {
			case watcher.CreateEvent:
				// TODO: create route
				logger.Infof("create (%s)", filename)
				runner.send(codegen.UpdateFileMessage(filename, []byte{}))

			case watcher.UpdateEvent:
				// TODO: update route
				logger.Infof("update (%s)", filename)
				runner.send(codegen.UpdateFileMessage(filename, []byte{}))

			case watcher.DeleteEvent:
				logger.Infof("delete (%s)", filename)
				runner.send(codegen.DeleteFileMessage(filename))
			}

			return nil
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
	err = c.GenerateProductionServer(routes)
	if err != nil {
		return err
	}

	return nil
}
