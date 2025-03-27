# ROADMAP

## Router

[ ] Modify the system to look for `.go` files inside `internal/www/` folder.
- Find `.go` inside subfolders

[ ] Use `//nova:route` directive to register HTTP function handlers
```
//nova:route post /users
func PostUsers(w http.ResponseWriter, r *http.Request) {}

//nova:route GET /users
func GetUsers(w http.ResponseWriter, r *http.Request) {}
```

[ ] Use `//nova:route` directive to register HTTP handlers
```
//nova:route get /users
type Users struct{}

func (*Users) ServeHTTP(w http.ResponseWriter, r *http.Request) {}
```

[ ] Dependency injection `//nova:inject` & `//nova:injectable`
```
//nova:injectable db
func NewDB() *sql.DB {
    // Lógica de conexión
    return &sql.DB{}
}

type Users struct {
    db *sql.DB `inject:"db"` // Inyección de dependencia
}

//nova:route GET /users/{id}
func (u *Users) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Usar u.db
}
```

[ ] Find `.html` files inside `src` folder

[ ] Use filesystem router for HTML files
```
src/index.html -> /
src/index.html -> /index.html
src/about.html -> /about
src/about.html -> /about.html
src/about/index.html -> /about/
src/about/index.html -> /about/index.html
```

[ ] Extract JS and CSS files from HTML files to be used as esbuild entries

## Development Server

[X] In-memory filesystem

[X] In-memory router

[ ] Watch `internal/www/` to update `.go` routes
- Find created/updated/deleted files
- Update in-memory router based on events

[ ] Enable esbuild watch-mode for `src` folder
- Handle OnEnd callback to create/update/delete in-memory files
- Clean-up the in-memory filesystem first (fastest)
- Refactor to update in-memory filesystem real-changes only

[ ] Hot Module Replacement & Hot Reload
- Create `main.go` files inside `.nova/internal` for every route
- `main.go` imports user's files from `internal/www/`
- In-memory router maps request to `main.go` files by using `mux.Handle`
- `hmr.js` opens an `EventSource` to listen for server events
- `.go` route created and deleted changes make GET request to reload `location.reload`
- `.js` changes make scripts to be replaced
- `.css` changes make links to be replaced

## Production Build

[ ] Minify and copy `.html` files to `.nova/pages/`

[ ] Use esbuild minifier for `.js` & `.css` copy them to `.nova/static/`

[ ] Code generation for `.nova/main.go`
- Use `embed` for `.nova/pages/`
- Use `embed` for `.nova/static/`
- Import user's routes from `internal/www/`

## Config

[ ] Server-side routes folder. Default: `internal/www`

[ ] Static routes folder. Default: `src`

[ ] Dev-server host. Default: `0.0.0.0`

[ ] Dev-server port. Default: `8080`

[ ] Watcher scan (create, delete) interval. Default: `250ms`

[ ] Watcher update interval. Default `250ms`

[ ] Output directory. Default `.nova`
