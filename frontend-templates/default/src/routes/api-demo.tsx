import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { api } from "../lib/api-client";
import type { Post, User } from "../types/generated";

export const Route = createFileRoute("/api-demo")({
  component: ApiDemoPage,
});

function ApiDemoPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [posts, setPosts] = useState<Post[]>([]);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [selectedPost, setSelectedPost] = useState<Post | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>("");
  const [newUser, setNewUser] = useState({
    name: "",
    email: "",
    password: "",
    role: "",
  });
  const [newPost, setNewPost] = useState({
    title: "",
    content: "",
    author_id: 1,
  });

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    setError("");

    try {
      const [usersResult, postsResult] = await Promise.all([
        api.users.list(),
        api.posts.list(),
      ]);

      if (!usersResult.success) {
        setError(usersResult.error.detail);
      } else {
        setUsers(usersResult.data || []);
      }

      if (!postsResult.success) {
        setError(postsResult.error.detail);
      } else {
        setPosts(postsResult.data || []);
      }
    } catch (err) {
      setError("Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  const formatError = (errorString: string) => {
    try {
      const errorObj = JSON.parse(errorString);
      if (errorObj.details && Array.isArray(errorObj.details)) {
        return (
          <div>
            <div className="mb-2 font-medium">{errorObj.error}</div>
            <ul className="space-y-1 list-disc list-inside">
              {errorObj.details.map((detail: any, index: number) => (
                <li key={index} className="text-sm">
                  <span className="font-medium">{detail.field}:</span>{" "}
                  {detail.message}
                  {detail.value !== undefined && detail.value !== null && (
                    <span className="text-gray-600">
                      {" "}
                      (received: {JSON.stringify(detail.value)})
                    </span>
                  )}
                </li>
              ))}
            </ul>
          </div>
        );
      }
      return errorObj.error || errorString;
    } catch {
      return errorString;
    }
  };

  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    const result = await api.users.create({
      name: newUser.name,
      email: newUser.email,
      age: 0,
    });

    if (!result.success) {
      setError(result.error.detail);
    } else {
      setNewUser({ name: "", email: "", password: "", role: "" });
      await loadData();
    }
    setLoading(false);
  };

  const handleCreatePost = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    const result = await api.posts.create({
      title: newPost.title,
      content: newPost.content,
      published: true,
      user_id: newPost.author_id,
    });

    if (!result.success) {
      setError(result.error.detail);
    } else {
      setNewPost({ title: "", content: "", author_id: 1 });
      await loadData();
    }
    setLoading(false);
  };

  const handleGetUser = async (id: number) => {
    setLoading(true);
    setError("");

    const result = await api.users.get(id);
    if (!result.success) {
      setError(result.error.detail);
    } else {
      setSelectedUser(result.data || null);
    }
    setLoading(false);
  };

  const handleGetPost = async (id: number) => {
    setLoading(true);
    setError("");

    const result = await api.posts.get(id);
    if (!result.success) {
      setError(result.error.detail);
    } else {
      setSelectedPost(result.data || null);
    }
    setLoading(false);
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <h1 className="mb-4 text-3xl font-bold text-gray-900">
          API Demo - Auto-Generated Types
        </h1>
        <p className="max-w-3xl mx-auto text-lg text-gray-600">
          This page demonstrates the auto-generated TypeScript API client and
          types. All API calls are fully type-safe with IntelliSense support.
        </p>
        <div className="mt-4">
          <a
            href="/api/docs"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            📚 View API Documentation (Swagger)
            <svg
              className="w-4 h-4 ml-2"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
              />
            </svg>
          </a>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="p-4 mb-6 border border-red-200 rounded-md bg-red-50">
          <div className="text-red-700">{formatError(error)}</div>
        </div>
      )}

      {loading && (
        <div className="p-4 mb-6 border border-blue-200 rounded-md bg-blue-50">
          <div className="text-blue-700">Loading...</div>
        </div>
      )}

      {/* Type Information */}
      <div className="p-6 border border-blue-200 rounded-lg bg-blue-50">
        <h2 className="mb-4 text-xl font-semibold text-gray-900">
          Generated Types
        </h2>
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div>
            <h3 className="mb-2 font-medium text-gray-800">User Interface</h3>
            <pre className="p-3 overflow-x-auto text-sm bg-white rounded">
              {`interface User {
  id: number
  name: string
  email: string
}`}
            </pre>
          </div>
          <div>
            <h3 className="mb-2 font-medium text-gray-800">Post Interface</h3>
            <pre className="p-3 overflow-x-auto text-sm bg-white rounded">
              {`interface Post {
  id: number
  title: string
  content: string
  author_id: number
  created_at: string
}`}
            </pre>
          </div>
        </div>
      </div>

      {/* API Methods */}
      <div className="p-6 border border-green-200 rounded-lg bg-green-50">
        <h2 className="mb-4 text-xl font-semibold text-gray-900">
          Available API Methods
        </h2>
        <div className="grid grid-cols-1 gap-4 text-sm md:grid-cols-2">
          <div className="space-y-2">
            <h3 className="font-medium text-gray-800">User Methods</h3>
            <div className="p-3 space-y-1 font-mono bg-white rounded">
              <div>api.users.list(): Promise&lt;User[]&gt;</div>
              <div>api.users.get(id): Promise&lt;User&gt;</div>
              <div>api.users.create(data): Promise&lt;User&gt;</div>
            </div>
          </div>
          <div className="space-y-2">
            <h3 className="font-medium text-gray-800">Post Methods</h3>
            <div className="p-3 space-y-1 font-mono bg-white rounded">
              <div>api.posts.list(): Promise&lt;Post[]&gt;</div>
              <div>api.posts.get(id): Promise&lt;Post&gt;</div>
              <div>api.posts.create(data): Promise&lt;Post&gt;</div>
            </div>
          </div>
        </div>
      </div>

      {/* Interactive Demo */}
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
        {/* Users Section */}
        <div className="space-y-6">
          <div className="p-6 bg-white rounded-lg shadow">
            <h2 className="mb-4 text-xl font-semibold text-gray-900">Users</h2>

            {/* Create User Form */}
            <form
              onSubmit={handleCreateUser}
              className="p-4 mb-6 rounded bg-gray-50"
            >
              <h3 className="mb-3 font-medium text-gray-800">
                Create New User
              </h3>
              <div className="space-y-3">
                <input
                  type="text"
                  placeholder="Name"
                  value={newUser.name}
                  onChange={(e) =>
                    setNewUser({ ...newUser, name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                />
                <input
                  type="email"
                  placeholder="Email"
                  value={newUser.email}
                  onChange={(e) =>
                    setNewUser({ ...newUser, email: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                />
                <input
                  type="password"
                  placeholder="Password (min 6 characters)"
                  value={newUser.password}
                  onChange={(e) =>
                    setNewUser({ ...newUser, password: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                  minLength={6}
                />
                <select
                  value={newUser.role}
                  onChange={(e) =>
                    setNewUser({ ...newUser, role: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="">User (default)</option>
                  <option value="user">User</option>
                  <option value="admin">Admin</option>
                </select>
                <button
                  type="submit"
                  disabled={loading}
                  className="w-full px-4 py-2 text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {loading ? "Creating..." : "Create User"}
                </button>
              </div>
            </form>

            {/* Users List */}
            <div className="space-y-2">
              <h3 className="font-medium text-gray-800">
                All Users ({users.length})
              </h3>
              {users.map((user) => (
                <div
                  key={user.id}
                  className="flex items-center justify-between p-3 rounded bg-gray-50"
                >
                  <div>
                    <div className="font-medium">{user.email}</div>
                    <div className="text-sm text-gray-600">{user.email}</div>
                  </div>
                  <button
                    onClick={() => handleGetUser(user.id)}
                    className="text-sm text-blue-600 hover:text-blue-800"
                  >
                    Get Details
                  </button>
                </div>
              ))}
            </div>

            {/* Selected User */}
            {selectedUser && (
              <div className="p-4 mt-4 border border-blue-200 rounded bg-blue-50">
                <h4 className="mb-2 font-medium text-gray-800">
                  Selected User (api.users.get)
                </h4>
                <pre className="text-sm">
                  {JSON.stringify(selectedUser, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>

        {/* Posts Section */}
        <div className="space-y-6">
          <div className="p-6 bg-white rounded-lg shadow">
            <h2 className="mb-4 text-xl font-semibold text-gray-900">Posts</h2>

            {/* Create Post Form */}
            <form
              onSubmit={handleCreatePost}
              className="p-4 mb-6 rounded bg-gray-50"
            >
              <h3 className="mb-3 font-medium text-gray-800">
                Create New Post
              </h3>
              <div className="space-y-3">
                <input
                  type="text"
                  placeholder="Title"
                  value={newPost.title}
                  onChange={(e) =>
                    setNewPost({ ...newPost, title: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  required
                />
                <textarea
                  placeholder="Content"
                  value={newPost.content}
                  onChange={(e) =>
                    setNewPost({ ...newPost, content: e.target.value })
                  }
                  className="w-full h-20 px-3 py-2 border border-gray-300 rounded-md"
                  required
                />
                <select
                  value={newPost.author_id}
                  onChange={(e) =>
                    setNewPost({
                      ...newPost,
                      author_id: parseInt(e.target.value),
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                >
                  {users.map((user) => (
                    <option key={user.id} value={user.id}>
                      {user.email}
                    </option>
                  ))}
                </select>
                <button
                  type="submit"
                  disabled={loading}
                  className="w-full px-4 py-2 text-white bg-green-600 rounded-md hover:bg-green-700 disabled:opacity-50"
                >
                  {loading ? "Creating..." : "Create Post"}
                </button>
              </div>
            </form>

            {/* Posts List */}
            <div className="space-y-2">
              <h3 className="font-medium text-gray-800">
                All Posts ({posts.length})
              </h3>
              {posts.map((post) => (
                <div key={post.id} className="p-3 rounded bg-gray-50">
                  <div className="flex items-center justify-between mb-2">
                    <div className="font-medium">{post.title}</div>
                    <button
                      onClick={() => handleGetPost(post.id)}
                      className="text-sm text-green-600 hover:text-green-800"
                    >
                      Get Details
                    </button>
                  </div>
                  <div className="text-sm text-gray-600">{post.content}</div>
                  <div className="mt-1 text-xs text-gray-500">
                    By{" "}
                    {users.find((u) => u.id === post.user_id)?.email ||
                      "Unknown"}
                  </div>
                </div>
              ))}
            </div>

            {/* Selected Post */}
            {selectedPost && (
              <div className="p-4 mt-4 border border-green-200 rounded bg-green-50">
                <h4 className="mb-2 font-medium text-gray-800">
                  Selected Post (api.posts.get)
                </h4>
                <pre className="text-sm">
                  {JSON.stringify(selectedPost, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Code Examples */}
      <div className="p-6 rounded-lg bg-gray-50">
        <h2 className="mb-4 text-xl font-semibold text-gray-900">
          Code Examples
        </h2>
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div>
            <h3 className="mb-2 font-medium text-gray-800">Fetching Data</h3>
            <pre className="p-3 overflow-x-auto text-sm bg-white rounded">
              {`// Type-safe API calls
const usersResult = await api.users.list()
if (usersResult.data) {
  setUsers(usersResult.data) // User[]
}

const userResult = await api.users.get(1)
if (userResult.data) {
  setUser(userResult.data) // User
}`}
            </pre>
          </div>
          <div>
            <h3 className="mb-2 font-medium text-gray-800">Creating Data</h3>
            <pre className="p-3 overflow-x-auto text-sm bg-white rounded">
              {`// Type-safe creation
const newUser: Omit<User, 'id'> = {
  name: 'John Doe',
  email: 'john@example.com'
}

const result = await api.users.create(newUser)
if (result.data) {
  console.log(result.data) // User with id
}`}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
}
