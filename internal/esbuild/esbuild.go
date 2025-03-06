package esbuild

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/module"
)

type ESBuildContext interface {
	Build() (map[string]string, error)

	Dispose()
}

type esbuildContextImpl struct {
	app         api.BuildContext
	nodeModules api.BuildContext
}

func (ctx *esbuildContextImpl) Build() (map[string]string, error) {
	result := ctx.nodeModules.Rebuild()
	if len(result.Errors) > 0 {

	}

	result = ctx.app.Rebuild()
	files := map[string]string{}
	for _, f := range result.OutputFiles {
		files[f.Path] = string(f.Contents)
	}

	return files, nil
}

func (ctx *esbuildContextImpl) Dispose() {
	ctx.nodeModules.Dispose()
	ctx.app.Dispose()
}

type ESBuild struct {
	config *config.Config
}

func NewESBuild(c *config.Config) *ESBuild {
	return &ESBuild{config: c}
}

func (esbuild *ESBuild) Context(entryPoints []string) ESBuildContext {
	outDir := module.Abs(filepath.Join(esbuild.config.Codegen.OutDir, "static"))

	nodeModules := map[string]string{}
	appCtx, ctxErr := api.Context(api.BuildOptions{
		EntryPoints: entryPoints,
		Outdir:      outDir,
		Format:      api.FormatESModule,
		Bundle:      true,
		Splitting:   true,
		Sourcemap:   api.SourceMapInline,
		Banner: map[string]string{
			"js": `import "/@nova/hmr.js";`,
		},
		Plugins: []api.Plugin{
			{
				Name: "nova-node_modules",
				Setup: func(pb api.PluginBuild) {
					pb.OnResolve(api.OnResolveOptions{Filter: "^[^./]"}, func(ora api.OnResolveArgs) (api.OnResolveResult, error) {
						packagepath := module.Abs(filepath.Join("node_modules", ora.Path))
						_, err := os.Stat(packagepath)
						switch {
						case err == os.ErrNotExist:
							return api.OnResolveResult{}, nil

						case err != nil:
							return api.OnResolveResult{}, err

						default:
						}

						if strings.HasSuffix(ora.ResolveDir, module.Abs("node_modules")) {
							return api.OnResolveResult{}, nil
						}

						result := pb.Resolve(ora.Path, api.ResolveOptions{
							ResolveDir: module.Abs("node_modules"),
							Importer:   ora.Importer,
							Namespace:  ora.Namespace,
							PluginData: ora.PluginData,
							With:       ora.With,
							Kind:       ora.Kind,
						})
						nodeModules[ora.Path] = result.Path

						return api.OnResolveResult{
							External:  true,
							Path:      "/@node_modules/" + ora.Path + ".js",
							Namespace: "@node_modules",

							PluginData: result.PluginData,
							Errors:     result.Errors,
							Warnings:   result.Warnings,
							Suffix:     result.Suffix,
						}, nil
					})
				},
			},
		},
	})
	if ctxErr != nil {
		log.Fatalln(ctxErr)
	}

	nodeModulesEntries := []api.EntryPoint{}
	for dep, file := range nodeModules {
		nodeModulesEntries = append(nodeModulesEntries, api.EntryPoint{
			InputPath:  file,
			OutputPath: dep,
		})
	}
	nodeModulesCtx, ctxErr := api.Context(api.BuildOptions{
		EntryPointsAdvanced: nodeModulesEntries,
		Bundle:              true,
		Write:               true,
		Outdir:              module.Abs(filepath.Join("node_modules", ".nova")),
		Format:              api.FormatESModule,
		MinifyWhitespace:    true,
		MinifyIdentifiers:   false,
		MinifySyntax:        true,
	})

	nodeModulesCtx.Rebuild()

	return &esbuildContextImpl{
		app:         appCtx,
		nodeModules: nodeModulesCtx,
	}
}
