import { createFileRoute } from "@tanstack/react-router";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Badge,
  Button,
} from "@/components/ui";

export const Route = createFileRoute("/about")({
  component: AboutPage,
});

function AboutPage() {
  return (
    <div className="space-y-12">
      {/* Hero Section */}
      <div className="text-center space-y-6">
        <div className="space-y-4">
          <Badge variant="brand" className="px-3 py-1">
            About GoFlux
          </Badge>
          <h1 className="text-4xl sm:text-5xl font-bold tracking-tight">
            Modern Fullstack{" "}
            <span className="bg-gradient-to-r from-brand-600 via-accent-600 to-brand-800 bg-clip-text text-transparent">
              Development
            </span>
          </h1>
          <p className="max-w-3xl mx-auto text-xl text-muted-foreground leading-relaxed">
            GoFlux combines the performance of Go with the developer experience
            of modern React, delivering type-safe fullstack applications with
            zero runtime dependencies.
          </p>
        </div>
      </div>

      {/* Tech Stack */}
      <div className="grid gap-8 md:grid-cols-2">
        <Card>
          <CardHeader>
            <div className="w-12 h-12 bg-gradient-to-br from-brand-500 to-brand-600 rounded-lg flex items-center justify-center mb-4">
              <svg
                className="w-6 h-6 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
                />
              </svg>
            </div>
            <CardTitle>Backend</CardTitle>
            <CardDescription>
              Powerful Go server, utilizing the Huma API framework, compatible
              with any Go router.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-2">
              <Badge variant="success">Go 1.22+</Badge>
              <span className="text-sm text-muted-foreground">
                Fast, concurrent, memory-safe
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="success">Huma API</Badge>
              <span className="text-sm text-muted-foreground">
                FastAPI-inspired API framework
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="success">GoFlux CLI</Badge>
              <span className="text-sm text-muted-foreground">
                Live reload for development
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="success">SQLC</Badge>
              <span className="text-sm text-muted-foreground">
                Generates Go structs from SQL queries
              </span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="w-12 h-12 bg-gradient-to-br from-accent-500 to-accent-600 rounded-lg flex items-center justify-center mb-4">
              <svg
                className="w-6 h-6 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                />
              </svg>
            </div>
            <CardTitle>Frontend</CardTitle>
            <CardDescription>Modern React with type safety</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-2">
              <Badge variant="brand">React 19</Badge>
              <span className="text-sm text-muted-foreground">
                Latest React features
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="brand">TanStack Router</Badge>
              <span className="text-sm text-muted-foreground">
                Type-safe routing
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="brand">React Query</Badge>
              <span className="text-sm text-muted-foreground">
                Powerful data fetching
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Key Features */}
      <div className="space-y-8">
        <div className="text-center">
          <h2 className="text-3xl font-bold tracking-tight mb-4">
            Key Features
          </h2>
          <p className="text-muted-foreground text-lg">
            Everything you need for modern fullstack development
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <Card className="text-center">
            <CardHeader>
              <div className="w-12 h-12 bg-gradient-to-br from-success-500 to-success-600 rounded-lg flex items-center justify-center mx-auto mb-4">
                <svg
                  className="w-6 h-6 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                  />
                </svg>
              </div>
              <CardTitle>Flexible Type Generation</CardTitle>
              <CardDescription>
                Multiple API client patterns: Basic fetch, Axios, and tRPC-like
                type-safe clients
              </CardDescription>
            </CardHeader>
          </Card>

          <Card className="text-center">
            <CardHeader>
              <div className="w-12 h-12 bg-gradient-to-br from-warning-500 to-warning-600 rounded-lg flex items-center justify-center mx-auto mb-4">
                <svg
                  className="w-6 h-6 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M13 10V3L4 14h7v7l9-11h-7z"
                  />
                </svg>
              </div>
              <CardTitle>Hot Reload</CardTitle>
              <CardDescription>
                Frontend + Backend live reload with orchestration
              </CardDescription>
            </CardHeader>
          </Card>

          <Card className="text-center">
            <CardHeader>
              <div className="w-12 h-12 bg-gradient-to-br from-danger-500 to-danger-600 rounded-lg flex items-center justify-center mx-auto mb-4">
                <svg
                  className="w-6 h-6 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z"
                  />
                </svg>
              </div>
              <CardTitle>Zero Runtime</CardTitle>
              <CardDescription>
                Single Go binary, no Node.js required in production
              </CardDescription>
            </CardHeader>
          </Card>
        </div>
      </div>

      {/* Architecture */}
      <Card>
        <CardHeader>
          <CardTitle className="text-2xl">Architecture</CardTitle>
          <CardDescription>
            How GoFlux delivers full-stack type safety
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-6 md:grid-cols-3">
            <div className="text-center space-y-3">
              <div className="w-16 h-16 bg-gradient-to-br from-brand-500 to-brand-600 rounded-lg flex items-center justify-center mx-auto">
                <span className="text-white font-bold text-xl">1</span>
              </div>
              <h3 className="font-semibold">Define Go Structs</h3>
              <p className="text-sm text-muted-foreground">
                Write your API endpoints and data models in Go
              </p>
            </div>

            <div className="text-center space-y-3">
              <div className="w-16 h-16 bg-gradient-to-br from-accent-500 to-accent-600 rounded-lg flex items-center justify-center mx-auto">
                <span className="text-white font-bold text-xl">2</span>
              </div>
              <h3 className="font-semibold">Auto-Generate Types</h3>
              <p className="text-sm text-muted-foreground">
                TypeScript interfaces and API client generated automatically
              </p>
            </div>

            <div className="text-center space-y-3">
              <div className="w-16 h-16 bg-gradient-to-br from-success-500 to-success-600 rounded-lg flex items-center justify-center mx-auto">
                <span className="text-white font-bold text-xl">3</span>
              </div>
              <h3 className="font-semibold">Use in React</h3>
              <p className="text-sm text-muted-foreground">
                Full IntelliSense and type safety in your React components
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* API Generation Options */}
      <Card>
        <CardHeader>
          <CardTitle className="text-2xl">API Generation Options</CardTitle>
          <CardDescription>
            Choose the API client pattern that fits your project
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-6 md:grid-cols-3">
            <Card className="border-2 border-brand-200 bg-brand-50">
              <CardHeader>
                <div className="flex items-center gap-2 mb-2">
                  <Badge variant="brand">Basic</Badge>
                  <span className="text-sm text-muted-foreground">Default</span>
                </div>
                <CardTitle className="text-lg">Fetch API</CardTitle>
                <CardDescription>
                  Simple, lightweight client using native fetch
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-2">
                <div className="text-sm">
                  <div className="font-medium mb-1">Features:</div>
                  <ul className="text-muted-foreground space-y-1">
                    <li>• Zero dependencies</li>
                    <li>• TypeScript interfaces</li>
                    <li>• Basic error handling</li>
                    <li>• Perfect for simple apps</li>
                  </ul>
                </div>
              </CardContent>
            </Card>

            <Card className="border-2 border-accent-200 bg-accent-50">
              <CardHeader>
                <div className="flex items-center gap-2 mb-2">
                  <Badge variant="warning">Axios</Badge>
                  <span className="text-sm text-muted-foreground">Popular</span>
                </div>
                <CardTitle className="text-lg">Axios Client</CardTitle>
                <CardDescription>
                  Feature-rich HTTP client with interceptors
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-2">
                <div className="text-sm">
                  <div className="font-medium mb-1">Features:</div>
                  <ul className="text-muted-foreground space-y-1">
                    <li>• Request/response interceptors</li>
                    <li>• Automatic JSON parsing</li>
                    <li>• Request/response transformation</li>
                    <li>• Built-in timeout support</li>
                  </ul>
                </div>
              </CardContent>
            </Card>

            <Card className="border-2 border-success-200 bg-success-50">
              <CardHeader>
                <div className="flex items-center gap-2 mb-2">
                  <Badge variant="success">tRPC-like</Badge>
                  <span className="text-sm text-muted-foreground">
                    Advanced
                  </span>
                </div>
                <CardTitle className="text-lg">Type-Safe RPC</CardTitle>
                <CardDescription>
                  End-to-end type safety with procedure calls
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-2">
                <div className="text-sm">
                  <div className="font-medium mb-1">Features:</div>
                  <ul className="text-muted-foreground space-y-1">
                    <li>• Full type inference</li>
                    <li>• Procedure-based API</li>
                    <li>• React Query integration</li>
                    <li>• Maximum type safety</li>
                  </ul>
                </div>
              </CardContent>
            </Card>
          </div>

          <div className="bg-muted/50 rounded-lg p-4">
            <div className="flex items-start gap-3">
              <div className="w-6 h-6 bg-brand-500 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
                <svg
                  className="w-3 h-3 text-white"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                    clipRule="evenodd"
                  />
                </svg>
              </div>
              <div>
                <div className="font-medium mb-1">Choose Your Style</div>
                <p className="text-sm text-muted-foreground">
                  All generation types produce the same TypeScript interfaces
                  from your Go structs. The difference is in the API client
                  pattern and developer experience. You can switch between them
                  at any time without changing your Go backend.
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* CTA */}
      <div className="text-center space-y-6">
        <h2 className="text-3xl font-bold tracking-tight">
          Ready to build something amazing?
        </h2>
        <p className="text-muted-foreground text-lg max-w-2xl mx-auto">
          Experience the future of fullstack development with GoFlux. Fast,
          type-safe, and production-ready.
        </p>
        <div className="flex flex-wrap justify-center gap-4">
          <Button variant="brand" size="lg" asChild>
            <a href="/api-demo">Try the Demo</a>
          </Button>
          <Button variant="outline" size="lg" asChild>
            <a href="/api/docs" target="_blank" rel="noopener noreferrer">
              API Documentation
            </a>
          </Button>
        </div>
      </div>
    </div>
  );
}
