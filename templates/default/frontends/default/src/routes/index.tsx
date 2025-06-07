import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { api } from "../lib/api-client";
import type { Post, User } from "../types/generated";

export const Route = createFileRoute("/")({
  component: HomePage,
});

function countRoutes(obj: any = api, count = 0): number {
  for (const key in obj) {
    if (typeof obj[key] === "function") {
      count++;
    } else if (typeof obj[key] === "object" && obj[key] !== null) {
      count = countRoutes(obj[key], count);
    }
  }
  return count;
}

function HomePage() {
  const [posts, setPosts] = useState<Post[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    setError("");

    try {
      const [postsResult, usersResult] = await Promise.all([
        api.posts.list(),
        api.users.list(),
      ]);

      if (!postsResult.success) {
        setError(postsResult.error.detail);
      } else {
        setPosts(postsResult.data.posts || []);
      }

      if (!usersResult.success) {
        setError(usersResult.error.detail);
      } else {
        setUsers(usersResult.data.users || []);
      }
    } catch (err) {
      setError("Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  const getUserName = (authorId: number) => {
    const user = users.find((u) => u.id === authorId);
    return user ? user.name : "Unknown";
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="text-lg text-gray-600">Loading...</div>
      </div>
    );
  }

  return (
    <div className="px-4 py-8 mx-auto space-y-12 max-w-7xl sm:px-6 lg:px-8">
      {/* Hero Section */}
      <div className="text-center">
        <h1 className="mb-6 text-5xl font-bold text-gray-900">
          Go + TanStack Router Fullstack
        </h1>
        <p className="max-w-4xl mx-auto text-xl leading-relaxed text-gray-600">
          Modern fullstack development with{" "}
          <strong>auto-generated TypeScript types</strong> from Go structs.
          Features <strong>Go + Fiber</strong> backend,{" "}
          <strong>React + TanStack Router</strong> frontend, and{" "}
          <strong>zero Node.js runtime</strong> in production.
        </p>
      </div>

      {/* Type Generation Showcase */}
      <div className="p-8 border border-blue-200 bg-gradient-to-r from-blue-50 to-indigo-50 rounded-xl">
        <h2 className="mb-6 text-2xl font-semibold text-center text-gray-900">
          🔧 Auto-Generated Type Safety
        </h2>
        <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
          <div>
            <h3 className="mb-4 text-lg font-semibold text-gray-800">
              Go Structs → TypeScript Types
            </h3>
            <div className="p-4 overflow-x-auto font-mono text-sm bg-gray-900 rounded-lg">
              <div className="text-green-400">// Generated from Go structs</div>
              <div className="mt-2 text-blue-400">
                interface <span className="text-yellow-300">User</span> {"{"}
              </div>
              <div className="ml-4 text-gray-300">
                <div>
                  id: <span className="text-orange-400">number</span>
                </div>
                <div>
                  name: <span className="text-orange-400">string</span>
                </div>
                <div>
                  email: <span className="text-orange-400">string</span>
                </div>
              </div>
              <div className="text-blue-400">{"}"}</div>
            </div>
          </div>
          <div>
            <h3 className="mb-4 text-lg font-semibold text-gray-800">
              API Routes → Client Methods
            </h3>
            <div className="p-4 overflow-x-auto font-mono text-sm bg-gray-900 rounded-lg">
              <div className="text-green-400">// Auto-generated API client</div>
              <div className="mt-2 space-y-1 text-gray-300">
                <div>
                  <span className="text-blue-400">api</span>.
                  <span className="text-yellow-300">users</span>.
                  <span className="text-purple-400">list</span>():{" "}
                  <span className="text-orange-400">User[]</span>
                </div>
                <div>
                  <span className="text-blue-400">api</span>.
                  <span className="text-yellow-300">users</span>.
                  <span className="text-purple-400">create</span>(
                  <span className="text-orange-400">
                    Omit&lt;User, 'id'&gt;
                  </span>
                  )
                </div>
                <div>
                  <span className="text-blue-400">api</span>.
                  <span className="text-yellow-300">posts</span>.
                  <span className="text-purple-400">list</span>():{" "}
                  <span className="text-orange-400">Post[]</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="p-4 border border-red-200 rounded-lg bg-red-50">
          <div className="font-medium text-red-700">Error: {error}</div>
          <button
            onClick={loadData}
            className="mt-2 font-medium text-red-600 underline hover:text-red-800"
          >
            Try again
          </button>
        </div>
      )}

      {/* Development Stats */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <div className="p-6 text-center bg-white border border-gray-100 shadow-lg rounded-xl">
          <div className="mb-2 text-4xl font-bold text-blue-600">
            {posts.length}
          </div>
          <div className="font-medium text-gray-700">API Posts</div>
          <div className="mt-1 text-sm text-gray-500">From Go backend</div>
        </div>
        <div className="p-6 text-center bg-white border border-gray-100 shadow-lg rounded-xl">
          <div className="mb-2 text-4xl font-bold text-green-600">
            {users.length}
          </div>
          <div className="font-medium text-gray-700">API Users</div>
          <div className="mt-1 text-sm text-gray-500">Type-safe calls</div>
        </div>
        <div className="p-6 text-center bg-white border border-gray-100 shadow-lg rounded-xl">
          <div className="mb-2 text-4xl font-bold text-purple-600">
            {countRoutes()}
          </div>
          <div className="font-medium text-gray-700">API Routes</div>
          <div className="mt-1 text-sm text-gray-500">Auto-discovered</div>
        </div>
        <div className="p-6 text-center bg-white border border-gray-100 shadow-lg rounded-xl">
          <div className="mb-2 text-4xl font-bold text-red-600">0</div>
          <div className="font-medium text-gray-700">Node.js Runtime</div>
          <div className="mt-1 text-sm text-gray-500">Production ready</div>
        </div>
      </div>

      {/* Recent Posts */}
      <div className="bg-white border border-gray-100 shadow-lg rounded-xl">
        <div className="px-8 py-6 border-b border-gray-200">
          <h2 className="text-2xl font-semibold text-gray-900">Recent Posts</h2>
          <p className="mt-1 text-gray-500">
            Fetched via auto-generated API client
          </p>
        </div>
        <div className="divide-y divide-gray-200">
          {posts.slice(0, 3).map((post) => (
            <div
              key={post.id}
              className="px-8 py-6 transition-colors hover:bg-gray-50"
            >
              <h3 className="mb-2 text-xl font-medium text-gray-900">
                {post.title}
              </h3>
              <p className="mb-3 leading-relaxed text-gray-600">
                {post.content}
              </p>
              <p className="text-sm text-gray-500">
                By{" "}
                <span className="font-medium">{getUserName(post.userId)}</span>{" "}
                • {new Date(post.createdAt).toLocaleDateString()}
              </p>
            </div>
          ))}
          {posts.length === 0 && (
            <div className="px-8 py-12 text-center text-gray-500">
              No posts available
            </div>
          )}
        </div>
      </div>

      {/* Architecture Features */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
        <div className="p-6 bg-white border border-gray-100 shadow-lg rounded-xl">
          <div className="mb-3 text-xl font-semibold text-gray-900">
            🔧 Auto-Generated
          </div>
          <p className="leading-relaxed text-gray-600">
            TypeScript types and API client generated from Go code analysis
          </p>
        </div>
        <div className="p-6 bg-white border border-gray-100 shadow-lg rounded-xl">
          <div className="mb-3 text-xl font-semibold text-gray-900">
            ⚡ Hot Reload
          </div>
          <p className="leading-relaxed text-gray-600">
            Frontend (Vite) + Backend (Air) with TypeScript orchestrator
          </p>
        </div>
        <div className="p-6 bg-white border border-gray-100 shadow-lg rounded-xl">
          <div className="mb-3 text-xl font-semibold text-gray-900">
            🚀 Production
          </div>
          <p className="leading-relaxed text-gray-600">
            Single Go binary with embedded assets, no Node.js required
          </p>
        </div>
        <div className="p-6 bg-white border border-gray-100 shadow-lg rounded-xl">
          <div className="mb-3 text-xl font-semibold text-gray-900">
            🎯 Type Safe
          </div>
          <p className="leading-relaxed text-gray-600">
            End-to-end type safety from Go structs to React components
          </p>
        </div>
      </div>

      {/* Development Workflow */}
      <div className="p-8 border border-gray-200 bg-gray-50 rounded-xl">
        <h2 className="mb-6 text-2xl font-semibold text-center text-gray-900">
          Development Workflow
        </h2>
        <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
          <div className="p-6 bg-white border border-gray-200 rounded-lg">
            <div className="mb-3 text-lg font-semibold text-gray-800">
              1. 🔧 Start Development
            </div>
            <code className="block px-3 py-2 mb-3 font-mono text-sm bg-gray-100 rounded">
              pnpm dev
            </code>
            <p className="leading-relaxed text-gray-600">
              Starts both frontend and backend with auto-reload
            </p>
          </div>
          <div className="p-6 bg-white border border-gray-200 rounded-lg">
            <div className="mb-3 text-lg font-semibold text-gray-800">
              2. ✏️ Edit Go Structs
            </div>
            <code className="block px-3 py-2 mb-3 font-mono text-sm bg-gray-100 rounded">
              internal/api/routes.go
            </code>
            <p className="leading-relaxed text-gray-600">
              Types auto-regenerate on Go server restart
            </p>
          </div>
          <div className="p-6 bg-white border border-gray-200 rounded-lg">
            <div className="mb-3 text-lg font-semibold text-gray-800">
              3. 🎯 Use in Frontend
            </div>
            <code className="block px-3 py-2 mb-3 font-mono text-sm bg-gray-100 rounded">
              api.users.list()
            </code>
            <p className="leading-relaxed text-gray-600">
              Full TypeScript IntelliSense and type checking
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
