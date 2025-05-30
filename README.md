# GoFlux CLI

🚀 **The fastest way to build full-stack applications with Go + TypeScript**

GoFlux is a CLI and a micro-framework that creates and manages full-stack projects with Go backend and modern TypeScript frontend frameworks.

## ✨ Features

- 🔥 **Zero Config**: Get started with one command
- 🎯 **Type Safe**: End-to-end type safety from Go → TypeScript  
- ⚡ **Fast Development**: Hot reload for both backend and frontend
- 🎨 **Modern Frontend**: Choose from TanStack Router, Next.js, or Vite+React
- 📦 **Single Binary**: No runtime dependencies in production
- 🛠️ **CLI Managed**: Everything managed by the flux CLI with optional micro-framework

## 🚀 Quick Start

### Installation

```bash
# Install GoFlux CLI
go install github.com/barisgit/goflux@latest # (coming soon)

# Or build from source
git clone https://github.com/barisgit/goflux
cd goflux
go build -o flux .
mv flux /usr/local/bin/
```

### Create a New Project

```bash
# Create new full-stack project
flux new my-app

# Navigate to project
cd my-app

# Start development servers
flux dev
```

That's it! Your app is running at:

- **Proxy**: http://localhost:3000
- **Frontend**: http://localhost:3001 (or next available port that is not in use)
- **Backend**: http://localhost:3002 (or next available port that is not in use)
- **API**: http://localhost:3002/api/health

*Note: The proxy is used to serve the frontend and backend in the same way they will be served in production. In development, a node process is used to server the frontend, while in production, the frontend is served by the backend.*

## 🏗️ Project Structure

```
my-app/
├── flux.yaml          # Project configuration
├── go.mod              # Go dependencies
├── cmd/
│   ├── server/         # Go backend server
├── internal/
│   ├── api/           # API routes
│   └── types/         # Go types
└── frontend/          # React/TypeScript frontend
    ├── src/
    │   ├── types/     # Generated TypeScript types
    │   └── lib/       # API client
    └── package.json
```

## 🎛️ Configuration (flux.yaml)

```yaml
name: my-app
frontend:
  framework: tanstack-router
  install_cmd: pnpm create @tanstack/router@latest frontend --typescript
  dev_cmd: cd frontend && pnpm dev
  build_cmd: cd frontend && pnpm build
  types_dir: src/types
  lib_dir: src/lib
backend:
  port: "3001"
development:
  type_gen_cmd: go run cmd/generate-types/main.go
```

## 📋 Available Commands

```bash
flux new <project-name>    # Create new project
flux dev                   # Start development servers
flux --help               # Show all commands
```

## 🎨 Frontend Options

**TanStack Router (Recommended)**
- File-based routing
- Type-safe navigation
- Best performance

**Next.js**
- App router
- Built-in optimizations
- Large ecosystem

...and more.

*The CLI will provide list of all available frontend options.*

## 🛠️ How It Works

1. **Project Creation**: `flux new` creates a complete project structure
2. **Auto Setup**: `flux dev` automatically installs dependencies on first run
3. **Development**: Manages both Go and frontend servers with one command
4. **Type Generation**: Automatically generates TypeScript types from Go structs

## 🎯 Why GoFlux?

| Traditional Stack | GoFlux |
|------------------|----------|
| Multiple CLIs | Single CLI |
| Complex setup | One command |
| Runtime deps | Zero deps |
| ~50MB memory | ~7MB memory |
| Multiple configs | One config file |

## 🤝 Contributing

This is the GoFlux CLI repository. The CLI creates and manages GoFlux projects but is completely separate from them.

## 📄 License

MIT License
