# GoFlux Router Adapters

This directory contains router-specific adapters for GoFlux static file serving, following the same pattern as HumaCLI's adapter system. This approach significantly reduces binary size by allowing users to import only the router they need.

## Architecture

- **`base/`** - Contains router-agnostic core logic (`ServeStaticFile`, `StaticConfig`, etc.)
- **`adapters/chi/`** - Chi router specific adapter
- **`adapters/fiber/`** - Fiber router specific adapter  
- **`adapters/gin/`** - Gin router specific adapter
- **`adapters/echo/`** - Echo router specific adapter
- **`adapters/nethttp/`** - Standard library net/http adapter (compatible with mux, gorilla mux, fasthttp)

## Binary Size Benefits

**Before (monolithic approach):**

```go
// This imports ALL router dependencies
import "github.com/barisgit/goflux"
```

**After (adapter approach):**

```go
// Only imports the specific router you need
import gofluxfiber "github.com/barisgit/goflux/adapters/fiber"
import "github.com/barisgit/goflux/goflux"
```

## Usage

### Chi Router

```go
import (
    gofluxchi "github.com/barisgit/goflux/adapters/chi"
    "github.com/barisgit/goflux/goflux"
)

router.Handle("/*", gofluxchi.StaticHandler(assets, goflux.StaticConfig{SPAMode: true}))
```

### Fiber Router

```go
import (
    gofluxfiber "github.com/barisgit/goflux/adapters/fiber"
    "github.com/barisgit/goflux/goflux"
)

app.Use("/*", gofluxfiber.StaticHandler(assets, goflux.StaticConfig{SPAMode: true}))
```

### Gin Router

```go
import (
    gofluxgin "github.com/barisgit/goflux/adapters/gin"
    "github.com/barisgit/goflux/goflux"
)

router.NoRoute(gofluxgin.StaticHandler(assets, goflux.StaticConfig{SPAMode: true}))
```

### Echo Router

```go
import (
    gofluxecho "github.com/barisgit/goflux/adapters/echo"
    "github.com/barisgit/goflux/goflux"
)

router.Any("/*", gofluxecho.StaticHandler(assets, goflux.StaticConfig{SPAMode: true}))
```

### Standard Library / Gorilla Mux / FastHTTP

```go
import (
    gofluxnethttp "github.com/barisgit/goflux/adapters/nethttp"
    "github.com/barisgit/goflux/goflux"
)

// Standard library mux
router.Handle("/", gofluxnethttp.StaticHandler(assets, goflux.StaticConfig{SPAMode: true}))

// Gorilla Mux
router.PathPrefix("/").Handler(gofluxnethttp.StaticHandler(assets, goflux.StaticConfig{SPAMode: true}))
```

## How It Works

1. **Core Logic**: All static file logic is in `goflux.ServeStaticFile()` - router agnostic
2. **Adapters**: Each adapter imports only its specific router and wraps the core logic
3. **Templates**: GoFlux templates conditionally import only the needed adapter
4. **Binary Size**: Go's linker only includes the imported router dependencies

This is the same pattern used by HumaCLI, allowing GoFlux to support 7+ routers without bloating the binary size.
