package project

import (
	"io/fs"
	"maps"
	"path/filepath"
	"slices"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/parser"
)

type scanner struct {
	config *config.Config

	goFiles       []string
	jsFiles       []string
	cssFiles      []string
	htmlFiles     []string
	templateFiles []string
	assetFiles    []string

	pages []string
}

func newScanner(c *config.Config) *scanner {
	return &scanner{config: c}
}

func (p *scanner) scan() error {
	err := p.findFiles(p.config.Router.Pages)
	if err != nil {
		return err
	}

	err = p.linkFiles()
	if err != nil {
		return err
	}

	err = p.findPageFiles()
	if err != nil {
		return err
	}

	return nil
}

func (p *scanner) findFiles(base string) error {
	pages := module.Abs(filepath.FromSlash(base))

	p.goFiles = []string{}
	p.jsFiles = []string{}
	p.cssFiles = []string{}
	p.htmlFiles = []string{}
	p.assetFiles = []string{}

	err := filepath.WalkDir(pages, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		switch filepath.Ext(path) {
		case ".go":
			p.goFiles = append(p.goFiles, path)

		case ".js", ".mjs", ".jsx", ".mjsx", ".ts", ".tsx":
			p.jsFiles = append(p.jsFiles, path)

		case ".css":
			p.cssFiles = append(p.cssFiles, path)

		case ".html":
			p.htmlFiles = append(p.htmlFiles, path)

		default:
			p.assetFiles = append(p.assetFiles, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *scanner) linkFiles() error {
	linkedFiles := map[string][]string{}

	for _, filename := range p.goFiles {
		imports, err := parser.ParseImportsGo(filename)
		if err != nil {
			return err
		}
		linkedFiles[filename] = imports
	}

	p.templateFiles = []string{}
	html := slices.Concat(slices.Collect(maps.Values(linkedFiles))...)
	for _, filename := range html {
		if filepath.Ext(filename) == ".html" {
			p.templateFiles = append(p.templateFiles, filename)
		}
	}

	for _, filename := range p.htmlFiles {
		imports, err := parser.ParseImportsHTML(filename)
		if err != nil {
			return err
		}
		linkedFiles[filename] = imports
	}

	return nil
}

func (p *scanner) findPageFiles() error {
	p.pages = slices.Clone(p.goFiles)

	type void struct{}
	templates := map[string]void{}
	imports := p.templateFiles
	for _, tmpl := range imports {
		templates[tmpl] = void{}
	}

	// static pages
	for _, filename := range p.htmlFiles {
		if _, ok := templates[filename]; !ok {
			p.pages = append(p.pages, filename)
		}
	}

	return nil
}
