# Nova

__A full-stack development tool for Go, inspired by modern frameworks like Astro, Next.js and SvelteKit.__

Nova combines Go's performance with modern frontend workflows (SSR, CSR, API routes and static site generation) into a single binary. No complex setup -- just write code and deploy.

## Features

- 🚀 `nova dev`: Start a development server with:
  - Hot Reloading for Go files.
  - Hot Module Replacements for JavaScript files.
  - Automatic JavaScript/TypeScript bundling via esbuild.
  - SSR for Go templates (`.go` + `.html`).
  - API routes (`.go` files with HTTP handlers).

- 📦 `nova build`: Generate a production-ready binary with;
  - Static asset bundling and minification.
  - Embedded templates and static files.
  - Optimized single-binary output.

- 🌐 __Zero-Config__: No need to install dependencies manually. Just focus on code.

## Getting Started

### Installation

```bash
go install github.com/sgq995/nova@latest
```

### Create a Project

```bash
mkdir my-awesome-project && cd my-awesome-project
god mod init my-awesome-project
mkdir -p src/pages
nova dev
```

### Project Structure

```
my-awesome-project/
├── src/
│   ├── pages/          # Route-based files (SSR, CSR, API, static)
│   │   ├── index.go    # SSR route (Go) + API handlers
│   │   ├── index.js    # CSR route (JavaScript)
|   |   ├── index.html  # Go template
│   │   └── about.html  # Static page
│   └── assets/         # Static files (CSS, images, etc.)
└── go.mod              # Go module file
```

### Usage Examples

1. SSR with Go

__File__: `src/pages/index.go`

```go
package pages

import (
  "html/template"
  "net/http"
)

//nova:template index.html

func Render(t *template.Template, w http.ResponseWriter, r *http.Request) error {
  data := map[string]string{"Title": "Home"}
  return t.Execute(w, data)
}
```

__Template__: `src/pages/index.html`

```html
<h1>{{.Title}}</h1>
<div id="root"></div>
<script type="module" src="index.js"></script>
```

__CSR with JavaScript__:

__File__: `src/pages/index.js`

```javascript
const root = document.getElementById('root')
const message = 'Hello from JavaScript!'
root.innerHTML = `<div>${message}</div>`
```

## Commands

### Development

```bash
nova dev
```

Starts a dev server with Hot Reloading and automatic asset bundling.


### Production Build

```bash
nova build
```

Generates a single binary in `.nova/` with embedded assets and optimized code.

## Roadmap (Future)

- 📦 __Plugins__: Extend Nova with support for React, Svelte, Vue, etc.
- 📝 __SSG__: Static Site Generation for blazing-fast performance.
- ⚡ __SSR__: JavaScript server routes or JavaScript templates for Go

## License

MIT License
