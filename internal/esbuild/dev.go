package esbuild

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/module"
)

type ESBuildContext struct {
	config *config.Config

	mu  sync.Mutex
	app api.BuildContext
	// nodeModules api.BuildContext
}

func NewESBuildContext(c *config.Config) *ESBuildContext {
	return &ESBuildContext{
		config: c,
	}
}

func (ctx *ESBuildContext) buildNodeModules(nodeModules map[string]string) error {
	nodeModulesEntries := []api.EntryPoint{}
	for dep, file := range nodeModules {
		nodeModulesEntries = append(nodeModulesEntries, api.EntryPoint{
			InputPath:  file,
			OutputPath: dep,
		})
	}
	result := api.Build(api.BuildOptions{
		EntryPointsAdvanced: nodeModulesEntries,
		Bundle:              true,
		Write:               true,
		Outdir:              module.Join("node_modules", ".nova"),
		Format:              api.FormatESModule,
		MinifyWhitespace:    true,
		MinifyIdentifiers:   true,
		MinifySyntax:        true,
	})
	if len(result.Errors) > 0 {
		logger.Errorf("%+v", esbuildError(result.Errors))
		return esbuildError(result.Errors)
	}
	return nil
}

// func (ctx *ESBuildContext) Build() (map[string][]byte, error) {
// 	result := ctx.nodeModules.Rebuild()
// 	if len(result.Errors) > 0 {
// 		return nil, esbuildError(result.Errors)
// 	}

// 	result = ctx.app.Rebuild()
// 	if len(result.Errors) > 0 {
// 		return nil, esbuildError(result.Errors)
// 	}

// 	// TODO: get outputs, look a entry points and inputs for each ouput
// 	//       link entry point to actual src/pages file, inputs are the dependencies
// 	// logger.Debugf("metafile: %+v\n", result.Metafile)

// 	files := map[string][]byte{}
// 	for _, f := range result.OutputFiles {
// 		files[f.Path] = f.Contents
// 	}

// 	return files, nil
// }

func (ctx *ESBuildContext) Watch() error {
	errs := []error{}
	errs = append(errs, ctx.app.Watch(api.WatchOptions{}))
	// errs = append(errs, ctx.nodeModules.Watch(api.WatchOptions{}))
	return errors.Join(errs...)
}

func (ctx *ESBuildContext) Dispose() {
	// ctx.nodeModules.Dispose()
	ctx.app.Dispose()
}

// func (ctx *ESBuildContext) Define(entryPoints []string) error {
// 	ctx.mu.Lock()
// 	defer ctx.mu.Unlock()

// 	outDir := module.Join(ctx.config.Codegen.OutDir, "static")
// 	nodeModules := map[string]string{}

// 	appCtx, ctxErr := api.Context(api.BuildOptions{
// 		EntryPoints: entryPoints,
// 		Outdir:      outDir,
// 		Format:      api.FormatESModule,
// 		Bundle:      true,
// 		Splitting:   true,
// 		Sourcemap:   api.SourceMapInline,
// 		Banner: map[string]string{
// 			"js": `import "/@nova/hmr.js";`,
// 		},
// 		Plugins: []api.Plugin{
// 			{
// 				Name: "nova-node_modules",
// 				Setup: func(pb api.PluginBuild) {
// 					pb.OnResolve(api.OnResolveOptions{Filter: "^[^./]"}, func(ora api.OnResolveArgs) (api.OnResolveResult, error) {
// 						packagepath := module.Join("node_modules", ora.Path)
// 						_, err := os.Stat(packagepath)
// 						switch {
// 						case err == os.ErrNotExist:
// 							return api.OnResolveResult{}, nil

// 						case err != nil:
// 							return api.OnResolveResult{}, err

// 						default:
// 						}

// 						if ora.ResolveDir == module.Abs("node_modules") {
// 							return api.OnResolveResult{}, nil
// 						}

// 						logger.Debugf("[esbuild] resolve (%s)", ora.Path)

// 						result := pb.Resolve(ora.Path, api.ResolveOptions{
// 							ResolveDir: module.Abs("node_modules"),
// 							Importer:   ora.Importer,
// 							Namespace:  ora.Namespace,
// 							PluginData: ora.PluginData,
// 							With:       ora.With,
// 							Kind:       ora.Kind,
// 						})
// 						nodeModules[ora.Path] = result.Path

// 						return api.OnResolveResult{
// 							External:  true,
// 							Path:      "/@node_modules/" + ora.Path + ".js",
// 							Namespace: "@node_modules",

// 							PluginData: result.PluginData,
// 							Errors:     result.Errors,
// 							Warnings:   result.Warnings,
// 							Suffix:     result.Suffix,
// 						}, nil
// 					})

// 					pb.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
// 						// TODO: verify node_modules have changed and rebuild them

// 						return api.OnEndResult{}, nil
// 					})
// 				},
// 			},
// 		},
// 	})
// 	if ctxErr != nil {
// 		return ctxErr
// 	}

// 	result := appCtx.Rebuild()
// 	if len(result.Errors) > 0 {
// 		return esbuildError(result.Errors)
// 	}

// 	nodeModulesEntries := []api.EntryPoint{}
// 	for dep, file := range nodeModules {
// 		nodeModulesEntries = append(nodeModulesEntries, api.EntryPoint{
// 			InputPath:  file,
// 			OutputPath: dep,
// 		})
// 	}
// 	nodeModulesCtx, ctxErr := api.Context(api.BuildOptions{
// 		EntryPointsAdvanced: nodeModulesEntries,
// 		Bundle:              true,
// 		Write:               true,
// 		Outdir:              module.Join("node_modules", ".nova"),
// 		Format:              api.FormatESModule,
// 		MinifyWhitespace:    true,
// 		MinifyIdentifiers:   true,
// 		MinifySyntax:        true,
// 	})
// 	if ctxErr != nil {
// 		return ctxErr
// 	}

// 	result = nodeModulesCtx.Rebuild()
// 	if len(result.Errors) > 0 {
// 		logger.Errorf("%+v", esbuildError(result.Errors))
// 	}

// 	ctx.app = appCtx
// 	ctx.nodeModules = nodeModulesCtx

// 	return nil
// }

func (ctx *ESBuildContext) Start(entryPoints []string, onEnd func(files map[string][]byte) error) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	outDir := module.Join(ctx.config.Codegen.OutDir, "static")
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
					nodeModules := map[string]string{}

					pb.OnResolve(api.OnResolveOptions{Filter: "^[^./]"}, func(ora api.OnResolveArgs) (api.OnResolveResult, error) {
						packagepath := module.Join("node_modules", ora.Path)
						_, err := os.Stat(packagepath)
						switch {
						case err == os.ErrNotExist:
							return api.OnResolveResult{}, nil

						case err != nil:
							return api.OnResolveResult{}, err

						default:
						}

						if ora.ResolveDir == module.Abs("node_modules") {
							return api.OnResolveResult{}, nil
						}

						logger.Debugf("[esbuild] resolve (%s)", ora.Path)

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

					pb.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
						// TODO: verify node_modules have changed and rebuild them
						ctx.buildNodeModules(nodeModules)

						return api.OnEndResult{}, nil
					})
				},
			},
			{
				Name: "nova-callback",
				Setup: func(pb api.PluginBuild) {
					pb.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
						files := map[string][]byte{}
						for _, file := range result.OutputFiles {
							filename, err := filepath.Rel(outDir, file.Path)
							if err != nil {
								return api.OnEndResult{}, err
							}
							files[filename] = file.Contents
						}
						onEnd(files)
						return api.OnEndResult{}, nil
					})
				},
			},
		},
	})
	if ctxErr != nil {
		return ctxErr
	}

	err := appCtx.Watch(api.WatchOptions{})
	if err != nil {
		appCtx.Dispose()
		return err
	}

	ctx.app = appCtx

	return nil
}
