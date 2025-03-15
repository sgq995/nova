package codegen

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/router"
	"github.com/sgq995/nova/internal/utils"
)

type handler struct {
	Render  *router.RenderRouteGo
	Rest    []*router.RestRouteGo
	Package string
	Path    string
}

func generateMain(c *config.Config, files map[string][]router.Route) error {
	imports := map[string]string{}
	handlers := map[string]handler{}
	pages := filepath.Join(module.Root(), c.Router.Pages)
	for filename, routes := range files {
		if len(routes) == 0 {
			continue
		}

		if filepath.Ext(filename) != ".go" {
			continue
		}

		base, _ := filepath.Rel(module.Root(), filepath.Dir(filename))
		base = filepath.ToSlash(base)
		alias := strings.ReplaceAll(base, "/", "")
		pkg := path.Join(module.ModuleName(), base)
		imports[alias] = pkg
		handlerPath, _ := filepath.Rel(pages, filename)
		handlerPath = filepath.Dir(handlerPath)

		h := handler{
			Rest:    make([]*router.RestRouteGo, 0),
			Package: alias,
			Path:    handlerPath,
		}

		for _, route := range routes {
			switch v := route.(type) {
			case *router.RenderRouteGo:
				h.Render = v

			case *router.RestRouteGo:
				h.Rest = append(h.Rest, v)
			}
		}

		handlers[filename] = h
	}

	isProd := os.Getenv("NOVA_ENV") == "production"
	pagespath := module.Abs(c.Router.Pages)
	outDir := filepath.Join(module.Root(), c.Codegen.OutDir)

	errs := []error{}
	if !isProd {
		hmrFiles := map[string][]handler{}
		aliases := map[string]string{}
		packages := map[string]string{}
		for filename, handler := range handlers {
			base := utils.Must(filepath.Rel(module.Root(), filepath.Dir(filename)))
			base = filepath.ToSlash(base)
			alias := strings.ReplaceAll(base, "/", "")
			pkg := path.Join(module.ModuleName(), base)
			imports[alias] = pkg

			basepath := utils.Must(filepath.Rel(pagespath, filepath.Dir(filename)))
			mainfile := filepath.Join(outDir, "pages", basepath, "main.go")

			aliases[mainfile] = alias
			packages[mainfile] = pkg

			handlers := hmrFiles[mainfile]
			handlers = append(handlers, handler)
			hmrFiles[mainfile] = handlers
		}

		t := generateHMRTemplate()
		for filename, handlers := range hmrFiles {
			os.MkdirAll(filepath.Dir(filename), 0755)
			file, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer file.Close()

			errs = append(errs, t.Execute(file, map[string]any{
				"Alias":    aliases[filename],
				"Package":  packages[filename],
				"Root":     pagespath,
				"Handlers": handlers,
			}))
		}
	}

	out := filepath.Join(outDir, "main.go")
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()

	var t *template.Template
	if isProd {
		t = generateProdServerTemplate()
	} else {
		t = generateDevServerTemplate()
	}
	errs = append(errs, t.Execute(file, map[string]any{
		"IsProd":      isProd,
		"Imports":     imports,
		"Root":        pagespath,
		"OutDir":      outDir,
		"Handlers":    handlers,
		"Host":        c.Server.Host,
		"Port":        c.Server.Port,
		"NodeModules": module.Join("node_modules", ".nova"),
	}))

	return errors.Join(errs...)
}

type Codegen struct {
	config *config.Config
}

func NewCodegen(c *config.Config) *Codegen {
	return &Codegen{config: c}
}

func (c *Codegen) Generate(files map[string][]router.Route) error {
	outDir := filepath.Join(module.Root(), c.config.Codegen.OutDir)
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}

	err = generateMain(c.config, files)
	if err != nil {
		return err
	}

	return nil
}
