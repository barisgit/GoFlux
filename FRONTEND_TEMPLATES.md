# Frontend Template System

GoFlux supports four types of frontend generation to give you maximum flexibility in how you create your frontend applications.

## 1. Hardcoded Templates (Built-in)

These are pre-configured templates built into GoFlux for popular frontend frameworks.

### Available Templates

- **tanstack-router**: TanStack Router with TypeScript and hot reload
- **nextjs**: Next.js with TypeScript, Tailwind CSS, and ESLint
- **vite-react**: Vite with React and TypeScript
- **vue**: Vue 3 with TypeScript and Vite
- **sveltekit**: SvelteKit with TypeScript
- **minimal**: Basic TypeScript setup with Vite and React

### Usage

**Interactive:**
```bash
flux new my-app
# Choose "Hardcoded Template (Built-in)"
# Select from available templates
```

**Command Line:**
```bash
flux new my-app --template hardcoded --template-source tanstack-router
```

**Configuration:**
```yaml
frontend:
  framework: tanstack-router
  template:
    type: hardcoded
    source: tanstack-router
```

## 2. Script Templates (pnpx/npm create)

Use existing package managers' create scripts to generate your frontend.

### Examples

- `pnpm create vue@latest . --typescript`
- `pnpm create svelte@latest . --typescript`
- `pnpm create next-app@latest . --typescript --tailwind`
- `pnpx create-react-app . --template typescript`

### Usage

**Interactive:**
```bash
flux new my-app
# Choose "Script Template (pnpx/npm create)"
# Enter your script command
```

**Command Line:**
```bash
flux new my-app --template script --template-source "pnpm create vue@latest . --typescript"
```

**Configuration:**
```yaml
frontend:
  framework: vue-custom
  template:
    type: script
    source: "pnpm create vue@latest . --typescript"
```

## 3. Custom Commands

Run any custom command to generate your frontend.

### Use Cases

- Custom build scripts
- Docker-based generation
- Multi-step setup processes
- Integration with company-specific tools

### Template Variables

- `{{frontend_path}}`: Path to the frontend directory
- `{{project_name}}`: Name of the project

### Usage

**Interactive:**
```bash
flux new my-app
# Choose "Custom Command"
# Enter your custom command
# Optionally specify working directory
```

**Command Line:**
```bash
flux new my-app --template custom --template-source "my-custom-setup.sh {{project_name}}"
```

**Configuration:**
```yaml
frontend:
  framework: custom
  template:
    type: custom
    command: "my-custom-setup.sh {{project_name}}"
    dir: "/path/to/scripts"  # optional working directory
```

## 4. Remote Templates (GitHub/Local)

Use templates from GitHub repositories or local directories.

### Template Structure

A remote template is a repository or directory containing:

1. **flux-template.yaml** - Template manifest (required)
2. **Template files** - Your frontend template files
3. **Template files with .tmpl extension** - Files processed as Go templates

### Template Manifest (flux-template.yaml)

```yaml
name: "My Custom Template"
description: "A custom React frontend with special configuration"
version: "1.0.0"
author: "Your Name"
framework: "react-custom"

# Commands for different phases
commands:
  install: "pnpm install"
  dev: "pnpm dev --port {{port}} --host"
  build: "pnpm build"

# Default variables available in templates
variables:
  theme_color: "#3b82f6"
  app_title: "{{ProjectName}}"

# File processing rules (optional)
files:
  - source: "package.json.tmpl"
    destination: "package.json"
    template: true
  
  - source: "src/App.tsx.tmpl"
    destination: "src/App.tsx"
    template: true
  
  - source: "public/favicon.ico"
    destination: "public/favicon.ico"
    template: false
```

### Template Variables

Templates have access to these variables:

**Built-in:**
- `{{ProjectName}}`: Project name
- `{{project_name}}`: Project name (lowercase)
- `{{PROJECT_NAME}}`: Project name (uppercase)

**Custom:**
- Any variables defined in the `variables` section
- Any variables passed via the `vars` configuration

### Usage

**Interactive:**
```bash
flux new my-app
# Choose "Remote Template (GitHub/Local)"
# Enter GitHub URL or local path
# Choose version/branch
# Enable/disable caching
```

**Command Line:**
```bash
# GitHub repository
flux new my-app --template remote --template-source "https://github.com/user/my-template"

# Local directory
flux new my-app --template remote --template-source "/path/to/my/template"
```

**Configuration:**
```yaml
frontend:
  framework: remote-template
  template:
    type: remote
    url: "https://github.com/user/my-template"
    version: "v1.0.0"  # branch, tag, or "latest"
    cache: true
    vars:
      theme_color: "#ff6b6b"
      custom_var: "value"
```

### GitHub Integration

GoFlux automatically converts GitHub repository URLs to download URLs:

- `https://github.com/user/repo` → Downloads main branch
- Supports branches, tags, and commits
- Caches templates locally in `~/.flux/templates/`

### Local Templates

Point to any local directory containing a `flux-template.yaml`:

```yaml
frontend:
  template:
    type: remote
    url: "./my-local-template"
    cache: false  # No need to cache local templates
```

## Configuration Examples

### Complete flux.yaml Examples

**TanStack Router (Hardcoded):**
```yaml
name: my-app
frontend:
  framework: tanstack-router
  template:
    type: hardcoded
    source: tanstack-router
  dev_cmd: "cd frontend && pnpm dev --port {{port}} --host"
  build_cmd: "cd frontend && pnpm build"
```

**Vue with Custom Script:**
```yaml
name: my-app
frontend:
  framework: vue
  template:
    type: script
    source: "pnpm create vue@latest . --typescript --router --pinia"
  dev_cmd: "cd frontend && pnpm dev --port {{port}} --host"
  build_cmd: "cd frontend && pnpm build"
```

**Remote GitHub Template:**
```yaml
name: my-app
frontend:
  framework: custom-react
  template:
    type: remote
    url: "https://github.com/company/react-template"
    version: "v2.1.0"
    cache: true
    vars:
      api_url: "http://localhost:3000"
      theme: "dark"
  dev_cmd: "cd frontend && pnpm dev --port {{port}} --host"
  build_cmd: "cd frontend && pnpm build"
```

## Creating Your Own Remote Template

1. **Create a new repository or directory**

2. **Add flux-template.yaml:**
```yaml
name: "My Awesome Template"
description: "Custom frontend template with special sauce"
version: "1.0.0"
author: "Your Name"
framework: "react-awesome"

commands:
  install: "pnpm install"
  dev: "pnpm dev --port {{port}} --host"
  build: "pnpm build"

variables:
  app_title: "{{ProjectName}}"
  description: "Built with GoFlux"
```

3. **Add your template files:**
```
my-template/
├── flux-template.yaml
├── package.json.tmpl
├── index.html.tmpl
├── src/
│   ├── App.tsx.tmpl
│   └── main.tsx
├── public/
│   └── favicon.ico
└── README.md.tmpl
```

4. **Use Go template syntax in .tmpl files:**
```json
// package.json.tmpl
{
  "name": "{{.project_name}}-frontend",
  "description": "{{.description}}",
  "version": "1.0.0"
}
```

5. **Test your template:**
```bash
flux new test-app --template remote --template-source "./my-template"
```

6. **Publish and use:**
```bash
# Push to GitHub
git remote add origin https://github.com/user/my-template
git push -u origin main

# Use in projects
flux new my-app --template remote --template-source "https://github.com/user/my-template"
```

## Advanced Features

### Template Inheritance

Templates can reference other templates or extend existing ones by copying and modifying their `flux-template.yaml`.

### Conditional File Processing

Use Go template conditions in your `.tmpl` files:

```html
<!-- index.html.tmpl -->
<title>{{.app_title}}{{if .environment}} - {{.environment}}{{end}}</title>
```

### Custom File Extensions

Files ending in `.tmpl` are automatically processed as templates. Other files are copied as-is unless explicitly configured in the `files` section.

### Environment-Specific Templates

Use different templates for different environments:

```yaml
frontend:
  template:
    type: remote
    url: "https://github.com/company/templates"
    version: "{{.environment}}"  # dev, staging, prod branches
```

## Troubleshooting

### Template Not Found
- Verify the URL or path is correct
- Check if `flux-template.yaml` exists in the template
- Ensure you have network access for remote templates

### Template Variables Not Working
- Check variable names match exactly (case-sensitive)
- Verify `.tmpl` files use correct Go template syntax
- Use `--debug` flag to see template processing details

### Caching Issues
- Clear cache: `rm -rf ~/.flux/templates/`
- Disable caching: set `cache: false` in template config
- Force refresh by changing the version

### Permission Errors
- Ensure write permissions in project directory
- Check if template files have correct permissions
- For local templates, verify directory access

## Migration from Legacy System

If you have existing projects using the old `install_cmd` system, they will continue to work. GoFlux automatically detects and uses:

1. New `template` configuration if present
2. Legacy `install_cmd` as a script template
3. Framework-based hardcoded template as fallback

To migrate, update your `flux.yaml`:

**Before:**
```yaml
frontend:
  framework: tanstack-router
  install_cmd: "pnpx create-tsrouter-app@latest . --template file-router"
```

**After:**
```yaml
frontend:
  framework: tanstack-router
  template:
    type: hardcoded
    source: tanstack-router
```

This provides better IDE support, validation, and additional features while maintaining backward compatibility. 