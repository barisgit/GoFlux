import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/about")({
  component: AboutPage,
});

function AboutPage() {
  return (
    <div className="space-y-8">
      <h1 className="text-3xl font-bold text-gray-900">
        About Go + TanStack Router Fullstack
      </h1>

      <div className="prose max-w-none">
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-2xl font-semibold mb-4">Project Overview</h2>

          <p className="text-gray-700 mb-4">
            This project demonstrates a modern fullstack development workflow
            with
            <strong> auto-generated TypeScript types</strong> from Go structs,
            eliminating the need for manual type synchronization between
            frontend and backend.
          </p>

          <h3 className="text-xl font-semibold mt-6 mb-3">
            🔧 Auto-Generation Features
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-4">
            <div className="border border-gray-200 rounded-lg p-4">
              <h4 className="font-semibold text-lg mb-2">Type Generation</h4>
              <ul className="text-sm text-gray-600 space-y-1">
                <li>• Go structs → TypeScript interfaces</li>
                <li>• API routes → Client methods</li>
                <li>• Automatic type inference from handlers</li>
                <li>• Support for arrays, pointers, and time.Time</li>
                <li>• JSON tag mapping for field names</li>
              </ul>
            </div>

            <div className="border border-gray-200 rounded-lg p-4">
              <h4 className="font-semibold text-lg mb-2">
                Development Workflow
              </h4>
              <ul className="text-sm text-gray-600 space-y-1">
                <li>• TypeScript orchestrator for process management</li>
                <li>• Hot reload for both frontend and backend</li>
                <li>• Automatic type regeneration on Go changes</li>
                <li>• Colored, filtered logs for better DX</li>
                <li>• Single command development setup</li>
              </ul>
            </div>
          </div>

          <h3 className="text-xl font-semibold mt-6 mb-3">
            🏗️ Architecture Stack
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-4">
            <div className="border border-gray-200 rounded-lg p-4">
              <h4 className="font-semibold text-lg mb-2">🔧 Backend</h4>
              <ul className="text-sm text-gray-600 space-y-1">
                <li>• Go + Fiber for high-performance HTTP server</li>
                <li>• Air for hot reload during development</li>
                <li>• AST analysis for automatic type discovery</li>
                <li>• Development proxy for frontend integration</li>
                <li>• Production mode with embedded assets</li>
              </ul>
            </div>

            <div className="border border-gray-200 rounded-lg p-4">
              <h4 className="font-semibold text-lg mb-2">🎨 Frontend</h4>
              <ul className="text-sm text-gray-600 space-y-1">
                <li>• React 18+ with TanStack Router</li>
                <li>• Vite for fast development and building</li>
                <li>• Auto-generated TypeScript API client</li>
                <li>• File-based routing with type safety</li>
                <li>• Tailwind CSS for styling</li>
              </ul>
            </div>
          </div>

          <h3 className="text-xl font-semibold mt-6 mb-3">
            ⚡ Development Commands
          </h3>

          <div className="bg-gray-50 rounded-lg p-4 space-y-3">
            <div>
              <code className="bg-gray-200 px-2 py-1 rounded text-sm">
                pnpm dev
              </code>
              <span className="ml-3 text-sm text-gray-600">
                Start full development environment
              </span>
            </div>
            <div>
              <code className="bg-gray-200 px-2 py-1 rounded text-sm">
                go run cmd/generate-types/main.go
              </code>
              <span className="ml-3 text-sm text-gray-600">
                Regenerate TypeScript types manually
              </span>
            </div>
            <div>
              <code className="bg-gray-200 px-2 py-1 rounded text-sm">
                pnpm build
              </code>
              <span className="ml-3 text-sm text-gray-600">
                Build for production
              </span>
            </div>
            <div>
              <code className="bg-gray-200 px-2 py-1 rounded text-sm">
                go build main.go
              </code>
              <span className="ml-3 text-sm text-gray-600">
                Build Go binary for deployment
              </span>
            </div>
          </div>

          <h3 className="text-xl font-semibold mt-6 mb-3">🎯 Key Benefits</h3>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4">
            <div className="text-center p-4 bg-blue-50 rounded-lg">
              <div className="text-2xl mb-2">🔧</div>
              <div className="font-semibold">Auto-Generated</div>
              <div className="text-xs text-gray-600">No manual type sync</div>
            </div>
            <div className="text-center p-4 bg-green-50 rounded-lg">
              <div className="text-2xl mb-2">🔒</div>
              <div className="font-semibold">Type Safe</div>
              <div className="text-xs text-gray-600">End-to-end types</div>
            </div>
            <div className="text-center p-4 bg-purple-50 rounded-lg">
              <div className="text-2xl mb-2">⚡</div>
              <div className="font-semibold">Fast DX</div>
              <div className="text-xs text-gray-600">Hot reload both sides</div>
            </div>
            <div className="text-center p-4 bg-orange-50 rounded-lg">
              <div className="text-2xl mb-2">🚀</div>
              <div className="font-semibold">Production Ready</div>
              <div className="text-xs text-gray-600">Single binary deploy</div>
            </div>
          </div>

          <h3 className="text-xl font-semibold mt-6 mb-3">
            🔍 Type Generation Process
          </h3>

          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-700 space-y-2">
              <p>
                <strong>1. AST Analysis:</strong> The Go type generator analyzes
                your codebase using the Go AST parser to discover API routes and
                their associated types.
              </p>
              <p>
                <strong>2. Type Discovery:</strong> It examines function
                signatures, body parser calls, and JSON responses to infer
                request and response types automatically.
              </p>
              <p>
                <strong>3. TypeScript Generation:</strong> Go structs are
                converted to TypeScript interfaces with proper type mapping
                (time.Time → string, slices → arrays, etc.).
              </p>
              <p>
                <strong>4. API Client Generation:</strong> HTTP methods are
                generated with proper TypeScript signatures, including
                Omit&lt;T, 'id'&gt; for creation endpoints.
              </p>
            </div>
          </div>

          <h3 className="text-xl font-semibold mt-6 mb-3">
            📊 Performance Characteristics
          </h3>

          <div className="bg-gray-50 rounded-lg p-4">
            <div className="text-sm text-gray-700">
              <p className="mb-2">
                <strong>Development:</strong> TypeScript orchestrator manages
                both processes independently, preventing unnecessary frontend
                restarts when Go code changes.
              </p>
              <p className="mb-2">
                <strong>Type Generation:</strong> Fast AST analysis typically
                completes in under 100ms for medium-sized codebases.
              </p>
              <p>
                <strong>Production:</strong> Single Go binary with embedded
                frontend assets, no Node.js runtime required for deployment.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
