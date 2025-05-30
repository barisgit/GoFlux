import { Outlet, createRootRoute, Link } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'

export const Route = createRootRoute({
  component: () => (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center space-x-8">
              <h1 className="text-xl font-bold text-gray-900">Go + TanStack Router</h1>
              <div className="hidden md:block">
                <div className="ml-10 flex items-baseline space-x-4">
                  <Link
                    to="/"
                    className="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium [&.active]:text-blue-600 [&.active]:font-semibold"
                  >
                    Home
                  </Link>
                  <Link
                    to="/api-demo"
                    className="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium [&.active]:text-blue-600 [&.active]:font-semibold"
                  >
                    API Demo
                  </Link>
                  <Link
                    to="/posts"
                    className="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium [&.active]:text-blue-600 [&.active]:font-semibold"
                  >
                    Posts
                  </Link>
                  <Link
                    to="/users"
                    className="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium [&.active]:text-blue-600 [&.active]:font-semibold"
                  >
                    Users
                  </Link>
                  <Link
                    to="/about"
                    className="text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium [&.active]:text-blue-600 [&.active]:font-semibold"
                  >
                    About
                  </Link>
                </div>
              </div>
            </div>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <Outlet />
      </main>

      <TanStackRouterDevtools />
    </div>
  ),
})
