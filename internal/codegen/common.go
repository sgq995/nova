package codegen

const renderHandlerFunc string = `
func renderHandler(root string, templates []string, render func(*template.Template, http.ResponseWriter, *http.Request) error) http.Handler {
	{{- if .IsProd}}
	sub := must(fs.Sub(templatesFS, root))
	t := template.Must(template.ParseFS(sub, templates...))
	{{- end}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		{{if not .IsProd -}}
		fs := os.DirFS(filepath.Join("{{.Root}}", root))
		t := template.Must(template.ParseFS(fs, templates...)){{end}}
		err := render(t, w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
`
