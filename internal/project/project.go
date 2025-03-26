package project

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/sgq995/nova/internal/codegen"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/esbuild"
	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/router"
	"github.com/sgq995/nova/internal/server"
	"github.com/sgq995/nova/internal/watcher"
)

type Project interface {
	Dispose()
}

type ProjectContext interface {
	Serve(ctx context.Context) (Project, error)

	Build() error
}

func Context(c config.Config) (ProjectContext, error) {
	return &projectContextImpl{config: &c}, nil
}

type projectImpl struct {
	scanner *scanner
	router  *router.Router
	codegen *codegen.Codegen
	runner  *runner
	esbuild *esbuild.ESBuildContext
	server  *server.Server
}

func (p *projectImpl) Dispose() {
	p.esbuild.Dispose()
	// s.runner.stop()
	p.server.Close()
}

type projectContextImpl struct {
	config *config.Config
}

func (p *projectContextImpl) Serve(ctx context.Context) (Project, error) {
	os.Setenv("NOVA_ENV", "development")

	scanner := newScanner(p.config)
	e := esbuild.NewESBuildContext(p.config)
	r := router.NewRouter(p.config)
	c := codegen.NewCodegen(p.config)
	runner := newRunner(p.config)
	s := server.New(p.config)

	logger.Infof("starting nova dev server...")
	scanner.scan()

	static := slices.Concat(scanner.jsFiles, scanner.cssFiles)
	if err := e.Define(static); err != nil {
		return nil, err
	}

	project := &projectImpl{
		scanner: scanner,
		router:  r,
		codegen: c,
		runner:  runner,
		esbuild: e,
		server:  s,
	}

	// if err := c.GenerateDevelopmentServer(); err != nil {
	// 	return nil, err
	// }

	// files := map[string][]byte{"@nova/hmr.js": hmr}
	// if err := runner.start(files); err != nil {
	// 	return nil, err
	// }

	go watcher.WatchDir(ctx, p.config.Router.Pages, watcher.CallbackMap{
		"*.go": func(event watcher.Event, files []string) error {
			switch event {
			case watcher.CreateEvent, watcher.UpdateEvent:
				logger.Infof("%s %s", event, files)

				routesMap, err := project.router.ParseRoutes(files)
				if err != nil {
					return err
				}

				for _, filename := range files {
					err = project.codegen.GenerateRouteModule(filename, routesMap[filename])
					if err != nil {
						return err
					}
				}

				routes := slices.Concat(slices.Collect(maps.Values(routesMap))...)
				messages := []*server.Message{}

				for _, route := range routes {
					switch route := route.(type) {
					case *router.RenderRouteGo:
						messages = append(messages, server.CreateRouteMessage(route.Pattern))

					case *router.RestRouteGo:
						messages = append(messages, server.CreateRouteMessage(route.Pattern))
					}
				}

				if event == watcher.UpdateEvent {
					for _, filename := range files {
						messages = append(messages, server.UpdateFileMessage(filename, []byte{}))
					}
				}

				project.server.Send(server.BulkMessage(messages...))

			case watcher.DeleteEvent:
				logger.Infof("%s %s", event, files)

				messages := []*server.Message{}
				for _, filename := range files {
					routes := project.router.Routes[filename]
					for _, route := range routes {
						switch route := route.(type) {
						case *router.RenderRouteGo:
							messages = append(messages, server.DeleteRouteMessage(route.Pattern))

						case *router.RestRouteGo:
							messages = append(messages, server.DeleteRouteMessage(route.Pattern))
						}
					}
				}

				project.server.Send(server.BulkMessage(messages...))
			}

			return nil
		},
		// TODO: esbuild watch mode + on start callback
		"*.js,*.ts,*.css": func(event watcher.Event, files []string) error {
			logger.Infof("%s %s", event, files)

			if event == watcher.CreateEvent || event == watcher.DeleteEvent {
				err := scanner.scan()
				if err != nil {
					return err
				}
				static := slices.Concat(scanner.jsFiles, scanner.cssFiles)

				project.esbuild.Dispose()
				err = project.esbuild.Define(static)
				if err != nil {
					return err
				}
			}

			bundles, err := project.esbuild.Build()
			if err != nil {
				return err
			}

			messages := []*server.Message{}

			root := module.Abs(p.config.Router.Pages)
			for _, filename := range files {
				in, err := filepath.Rel(root, filename)
				if err != nil {
					return err
				}

				switch event {
				case watcher.CreateEvent, watcher.UpdateEvent:
					out := module.Join(p.config.Codegen.OutDir, "static", in)
					contents := bundles[out]
					messages = append(messages, server.UpdateFileMessage(in, contents))

				case watcher.DeleteEvent:
					messages = append(messages, server.DeleteFileMessage(in))
				}
			}

			project.server.Send(server.BulkMessage(messages...))

			return nil
		},
		"*.html": func(event watcher.Event, files []string) error {
			logger.Infof("%s %s", event, files)

			messages := []*server.Message{}

			for _, filename := range files {
				switch event {
				case watcher.CreateEvent, watcher.UpdateEvent:
					// TODO: create route
					messages = append(messages, server.UpdateFileMessage(filename, []byte{}))

				case watcher.DeleteEvent:
					messages = append(messages, server.DeleteFileMessage(filename))
				}
			}

			project.server.Send(server.BulkMessage(messages...))

			return nil
		},
	})

	go project.server.ListenAndServe()

	return project, nil
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
