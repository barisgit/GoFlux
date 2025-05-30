# Unified Template System

GoFlux now uses a unified template system where backend templates define which frontend options they support. This provides better integration and consistency between backend and frontend components.

## Template Structure

Each backend template is organized as follows:

```text
templates/
├── template-name/
│   ├── template.yaml          # Template configuration
│   ├── [backend files].tmpl   # Backend template files
│   └── frontends/             # Frontend template options
│       ├── default/           # Built-in frontend templates
│       │   ├── package.json.tmpl
│       │   ├── src/
│       │   └── ...
│       └── minimal/
│           ├── package.json.tmpl
│           └── ...
```

## Template Configuration (template.yaml)

Each template has a `template.yaml` file that defines:

```yaml
name: "Template Name"
description: "Template description"
version: "1.0.0"
backend:
  supported_routers:
    - "chi"
    - "fiber"
    - "gin"
    - "echo"
    - "gorilla"
  features:
    - "api"
    - "static_files"
    - "cors"

frontend:
  enable_script_frontends: true  # Allow script-based frontends
  options:
    - name: "default"
      description: "TanStack Router with React and Tailwind CSS"
      framework: "tanstack-router"
      dev_cmd: "cd frontend && pnpm dev --port {{port}} --host"
      build_cmd: "cd frontend && pnpm build"
      types_dir: "src/types"
      lib_dir: "src/lib"
      static_gen:
        enabled: false
    - name: "minimal"
      description: "Minimal TypeScript setup with basic React"
      framework: "minimal"
      dev_cmd: "cd frontend && pnpm dev --port {{port}} --host"
      build_cmd: "cd frontend && pnpm build"
      types_dir: "src/types"
      lib_dir: "src/lib"
```

## Frontend Generation Types

### 1. Template-based Frontends

Built-in frontend templates that are part of the backend template:

```yaml
frontend:
  template:
    type: "template"
    source: "default"  # Uses templates/default/frontends/default/
```

### 2. Script-based Frontends

Popular frontend frameworks using package manager scripts:

```yaml
frontend:
  template:
    type: "script"
    source: "vite-react"  # Uses registered script from script_registry.yaml
```

Or custom script:

```yaml
frontend:
  template:
    type: "script"
    source: "pnpm create vue@latest . -- --typescript"
```

### 3. Custom Commands

Any custom command for frontend generation:

```yaml
frontend:
  template:
    type: "custom"
    command: "my-custom-setup.sh {{project_name}}"
    dir: "/path/to/scripts"
```

## Script Registry

Script-based frontends are managed through `internal/frontend/script_registry.yaml`:

```yaml
categories:
  - name: "React"
    frameworks:
      - name: "vite-react"
        display_name: "Vite + React"
        description: "React with Vite bundler and TypeScript"
        script: "pnpm create vite@latest . -- --template react-ts"
        framework: "vite-react"
        dev_cmd: "cd frontend && pnpm dev --port {{port}} --host"
        build_cmd: "cd frontend && pnpm build"
```

## Template Variables

All templates have access to these variables:

- `{{.ProjectName}}`: Project name
- `{{.ModuleName}}`: Go module name
- `{{.GoVersion}}`: Go version
- `{{.BackendPort}}`: Backend server port
- `{{.Router}}`: Selected router (gorilla, chi, etc.)
- `{{.SPARouting}}`: Whether SPA routing is enabled
- `{{.ProjectDescription}}`: Template description
- `{{.CustomVars}}`: Additional custom variables

## Usage Examples

### Creating a Project with Template Frontend

```bash
flux new my-app --template default --frontend default
```

### Creating a Project with Script Frontend

```bash
flux new my-app --template default --frontend vite-react --frontend-type script
```

### Creating a Project with Custom Frontend

```bash
flux new my-app --template default --frontend-type custom --frontend-command "my-setup.sh"
```

## Adding New Backend Templates

1. Create template directory: `templates/my-template/`
2. Add `template.yaml` with configuration
3. Add backend template files (`.tmpl` extension)
4. Optionally add frontend templates in `frontends/` directory

## Adding New Script Frontends

Edit `internal/frontend/script_registry.yaml` and add your framework to the appropriate category.

## Migration from Legacy System

The new system is backward compatible. Existing projects using the old frontend template system will continue to work, but new projects will use the unified system for better integration and consistency. 