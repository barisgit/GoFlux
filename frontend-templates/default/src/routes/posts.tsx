import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { api } from "../lib/api-client";
import type { User, Post } from "../types/generated";

export const Route = createFileRoute("/posts")({
  component: PostsPage,
});

function PostsPage() {
  const [posts, setPosts] = useState<Post[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>("");
  const [showForm, setShowForm] = useState(false);
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
      const [postsResult, usersResult] = await Promise.all([
        api.posts.list(),
        api.users.list(),
      ]);

      if (!postsResult.success) {
        setError(postsResult.error.detail);
      } else {
        setPosts(postsResult.data || []);
      }

      if (!usersResult.success) {
        setError(usersResult.error.detail);
      } else {
        setUsers(usersResult.data || []);
      }
    } catch (err) {
      setError("Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  const handleCreatePost = async (e: React.FormEvent) => {
    e.preventDefault();

    const result = await api.posts.create({
      ...newPost,
      published: false,
      user_id: newPost.author_id,
    });
    if (!result.success) {
      setError(result.error.detail);
    } else {
      setNewPost({ title: "", content: "", author_id: 1 });
      setShowForm(false);
      loadData(); // Reload posts
    }
  };

  const getUserName = (authorId: number) => {
    const user = users.find((u) => u.id === authorId);
    return user ? user.email : "Unknown";
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="text-lg text-gray-600">Loading posts...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold text-gray-900">Posts</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 text-white bg-blue-600 rounded-md hover:bg-blue-700"
        >
          {showForm ? "Cancel" : "New Post"}
        </button>
      </div>

      {error && (
        <div className="p-4 border border-red-200 rounded-md bg-red-50">
          <div className="text-red-700">Error: {error}</div>
        </div>
      )}

      {showForm && (
        <div className="p-6 bg-white rounded-lg shadow">
          <h2 className="mb-4 text-xl font-semibold">Create New Post</h2>
          <form onSubmit={handleCreatePost} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">
                Title
              </label>
              <input
                type="text"
                value={newPost.title}
                onChange={(e) =>
                  setNewPost({ ...newPost, title: e.target.value })
                }
                className="block w-full mt-1 border-gray-300 rounded-md shadow-sm"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">
                Content
              </label>
              <textarea
                value={newPost.content}
                onChange={(e) =>
                  setNewPost({ ...newPost, content: e.target.value })
                }
                rows={4}
                className="block w-full mt-1 border-gray-300 rounded-md shadow-sm"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">
                Author
              </label>
              <select
                value={newPost.author_id}
                onChange={(e) =>
                  setNewPost({
                    ...newPost,
                    author_id: parseInt(e.target.value),
                  })
                }
                className="block w-full mt-1 border-gray-300 rounded-md shadow-sm"
              >
                {users.map((user) => (
                  <option key={user.id} value={user.id}>
                    {user.email}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex space-x-3">
              <button
                type="submit"
                className="px-4 py-2 text-white bg-green-600 rounded-md hover:bg-green-700"
              >
                Create Post
              </button>
              <button
                type="button"
                onClick={() => setShowForm(false)}
                className="px-4 py-2 text-white bg-gray-600 rounded-md hover:bg-gray-700"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="grid gap-6">
        {posts.map((post) => (
          <div key={post.id} className="p-6 bg-white rounded-lg shadow">
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <h2 className="mb-2 text-xl font-semibold text-gray-900">
                  {post.title}
                </h2>
                <p className="mb-4 text-gray-700">{post.content}</p>
                <div className="flex items-center text-sm text-gray-500">
                  <span>By {getUserName(post.user_id)}</span>
                  <span className="mx-2">•</span>
                </div>
              </div>
              <div className="flex space-x-2">
                <button className="text-sm text-blue-600 hover:text-blue-800">
                  Edit
                </button>
                <button className="text-sm text-red-600 hover:text-red-800">
                  Delete
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {posts.length === 0 && (
        <div className="py-12 text-center">
          <div className="text-lg text-gray-500">No posts yet</div>
          <p className="mt-2 text-gray-400">
            Create your first post to get started!
          </p>
        </div>
      )}
    </div>
  );
}
