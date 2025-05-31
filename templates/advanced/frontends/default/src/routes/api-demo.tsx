import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api-client";
import type { Post, User } from "../types/generated";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Button,
  Input,
  Textarea,
  Label,
  Badge,
  Alert,
  AlertDescription,
} from "@/components/ui";

export const Route = createFileRoute("/api-demo")({
  component: ApiDemoPage,
});

function ApiDemoPage() {
  const queryClient = useQueryClient();
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [selectedPost, setSelectedPost] = useState<Post | null>(null);
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null);
  const [selectedPostId, setSelectedPostId] = useState<number | null>(null);
  const [newUser, setNewUser] = useState({
    name: "",
    email: "",
    age: 0,
  });
  const [newPost, setNewPost] = useState({
    title: "",
    content: "",
    user_id: 1,
  });

  // Queries
  const usersQuery = useQuery(api.users.list.queryOptions());
  const postsQuery = useQuery(api.posts.list.queryOptions());

  // Mutations
  const createUserMutation = useMutation(
    api.users.create.mutationOptions({
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["users"] });
        setNewUser({ name: "", email: "", age: 0 });
      },
    })
  );

  const createPostMutation = useMutation(
    api.posts.create.mutationOptions({
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["posts"] });
        setNewPost({ title: "", content: "", user_id: 1 });
      },
    })
  );

  // Dynamic queries for selected user/post
  const getUserQuery = useQuery({
    ...api.users.get.queryOptions(selectedUserId || 1),
    enabled: selectedUserId !== null,
  });

  const getPostQuery = useQuery({
    ...api.posts.get.queryOptions(selectedPostId || 1),
    enabled: selectedPostId !== null,
  });

  // Update selected user/post when query data changes
  useEffect(() => {
    if (getUserQuery.data && selectedUserId !== null) {
      setSelectedUser(getUserQuery.data);
    }
  }, [getUserQuery.data, selectedUserId]);

  useEffect(() => {
    if (getPostQuery.data && selectedPostId !== null) {
      setSelectedPost(getPostQuery.data);
    }
  }, [getPostQuery.data, selectedPostId]);

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
                    <span className="text-muted-foreground">
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
    createUserMutation.mutate(newUser);
  };

  const handleCreatePost = async (e: React.FormEvent) => {
    e.preventDefault();
    createPostMutation.mutate({
      title: newPost.title,
      content: newPost.content,
      user_id: newPost.user_id,
      published: false,
    });
  };

  const handleGetUser = async (id: number) => {
    setSelectedUserId(id);
  };

  const handleGetPost = async (id: number) => {
    setSelectedPostId(id);
  };

  const isLoading =
    usersQuery.isLoading ||
    postsQuery.isLoading ||
    createUserMutation.isPending ||
    createPostMutation.isPending;

  const error =
    usersQuery.error ||
    postsQuery.error ||
    createUserMutation.error ||
    createPostMutation.error ||
    getUserQuery.error ||
    getPostQuery.error;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center space-y-6">
        <div className="space-y-4">
          <Badge variant="brand" className="px-3 py-1">
            Interactive Demo
          </Badge>
          <h1 className="text-3xl sm:text-4xl font-bold tracking-tight">
            API Demo -{" "}
            <span className="bg-gradient-to-r from-brand-600 via-accent-600 to-brand-800 bg-clip-text text-transparent">
              Auto-Generated Types
            </span>{" "}
            + React Query
          </h1>
          <p className="max-w-3xl mx-auto text-lg text-muted-foreground leading-relaxed">
            This page demonstrates the auto-generated TypeScript API client with
            React Query integration. All API calls are fully type-safe with
            IntelliSense support and automatic caching.
          </p>
        </div>

        <Button variant="outline" size="lg" asChild>
          <a
            href="/api/docs"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"
              />
            </svg>
            View API Documentation (Swagger)
          </a>
        </Button>
      </div>

      {/* Loading & Error States */}
      {isLoading && (
        <Alert variant="info">
          <AlertDescription>Loading data...</AlertDescription>
        </Alert>
      )}

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{formatError(error.message)}</AlertDescription>
        </Alert>
      )}

      {/* React Query Benefits */}
      <Card className="bg-gradient-to-br from-success-50 to-success-100 border-success-200">
        <CardHeader>
          <CardTitle className="text-xl text-success-900">
            React Query Integration Benefits
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div className="p-4 bg-white rounded-lg shadow-sm">
              <div className="flex items-center gap-2 mb-2">
                <div className="w-8 h-8 bg-success-500 rounded-lg flex items-center justify-center">
                  <svg
                    className="w-4 h-4 text-white"
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
                <h3 className="font-semibold text-success-900">
                  Automatic Caching
                </h3>
              </div>
              <p className="text-sm text-success-700">
                Data is cached automatically and shared across components
              </p>
            </div>
            <div className="p-4 bg-white rounded-lg shadow-sm">
              <div className="flex items-center gap-2 mb-2">
                <div className="w-8 h-8 bg-success-500 rounded-lg flex items-center justify-center">
                  <svg
                    className="w-4 h-4 text-white"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                    />
                  </svg>
                </div>
                <h3 className="font-semibold text-success-900">
                  Background Updates
                </h3>
              </div>
              <p className="text-sm text-success-700">
                Data stays fresh with background refetching
              </p>
            </div>
            <div className="p-4 bg-white rounded-lg shadow-sm">
              <div className="flex items-center gap-2 mb-2">
                <div className="w-8 h-8 bg-success-500 rounded-lg flex items-center justify-center">
                  <svg
                    className="w-4 h-4 text-white"
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
                <h3 className="font-semibold text-success-900">
                  Optimistic Updates
                </h3>
              </div>
              <p className="text-sm text-success-700">
                UI updates optimistically on mutations
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Type Information */}
      <Card className="bg-gradient-to-br from-brand-50 to-brand-100 border-brand-200">
        <CardHeader>
          <CardTitle className="text-xl text-brand-900">
            Generated Types
          </CardTitle>
          <CardDescription className="text-brand-700">
            Auto-generated TypeScript interfaces from Go structs
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div>
              <div className="flex items-center gap-2 mb-3">
                <Badge variant="brand">User Interface</Badge>
              </div>
              <div className="p-4 bg-white rounded-lg border border-brand-200 font-mono text-sm overflow-x-auto">
                <pre className="text-brand-800">{`interface User {
  id: number
  name: string
  email: string
  age?: number
  created_at?: string
  updated_at?: string
}`}</pre>
              </div>
            </div>
            <div>
              <div className="flex items-center gap-2 mb-3">
                <Badge variant="brand">Post Interface</Badge>
              </div>
              <div className="p-4 bg-white rounded-lg border border-brand-200 font-mono text-sm overflow-x-auto">
                <pre className="text-brand-800">{`interface Post {
  id: number
  title: string
  content: string
  user_id: number
  published: boolean
  created_at?: string
  updated_at?: string
}`}</pre>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Interactive Demo */}
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
        {/* Users Section */}
        <Card>
          <CardHeader>
            <CardTitle className="text-xl flex items-center gap-2">
              <div className="w-8 h-8 bg-brand-500 rounded-lg flex items-center justify-center">
                <svg
                  className="w-4 h-4 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
                  />
                </svg>
              </div>
              Users Management
            </CardTitle>
            <CardDescription>
              Create and manage users with full type safety
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Create User Form */}
            <Card className="bg-muted/50">
              <CardHeader>
                <CardTitle className="text-lg">Create New User</CardTitle>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleCreateUser} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="userName">Name</Label>
                    <Input
                      id="userName"
                      type="text"
                      placeholder="Enter full name"
                      value={newUser.name}
                      onChange={(e) =>
                        setNewUser({ ...newUser, name: e.target.value })
                      }
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="userEmail">Email</Label>
                    <Input
                      id="userEmail"
                      type="email"
                      placeholder="Enter email address"
                      value={newUser.email}
                      onChange={(e) =>
                        setNewUser({ ...newUser, email: e.target.value })
                      }
                      required
                    />
                  </div>
                  <Button
                    type="submit"
                    disabled={createUserMutation.isPending}
                    variant="success"
                    className="w-full"
                  >
                    {createUserMutation.isPending
                      ? "Creating..."
                      : "Create User"}
                  </Button>
                </form>
              </CardContent>
            </Card>

            {/* Users List */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="font-semibold">All Users</h3>
                <Badge variant="secondary">
                  {usersQuery.data?.length || 0} users
                </Badge>
              </div>
              <div className="max-h-64 overflow-y-auto space-y-2">
                {usersQuery.data?.map((user) => (
                  <div
                    key={user.id}
                    className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50 transition-colors"
                  >
                    <div className="flex-1">
                      <div className="font-medium">{user.name}</div>
                      <div className="text-sm text-muted-foreground">
                        {user.email}
                      </div>
                    </div>
                    <Button
                      onClick={() => handleGetUser(user.id)}
                      variant="outline"
                      size="sm"
                    >
                      View Details
                    </Button>
                  </div>
                ))}
              </div>
            </div>

            {/* Selected User Details */}
            {selectedUser && (
              <Card className="bg-gradient-to-br from-success-50 to-success-100 border-success-200">
                <CardHeader>
                  <CardTitle className="text-lg text-success-900">
                    Selected User Details
                    {getUserQuery.isPending && (
                      <Badge variant="secondary" className="ml-2">
                        Loading...
                      </Badge>
                    )}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="bg-white rounded-lg p-4 border border-success-200">
                    <pre className="text-sm text-success-800 overflow-x-auto">
                      {JSON.stringify(selectedUser, null, 2)}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            )}
          </CardContent>
        </Card>

        {/* Posts Section */}
        <Card>
          <CardHeader>
            <CardTitle className="text-xl flex items-center gap-2">
              <div className="w-8 h-8 bg-accent-500 rounded-lg flex items-center justify-center">
                <svg
                  className="w-4 h-4 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                  />
                </svg>
              </div>
              Posts Management
            </CardTitle>
            <CardDescription>
              Create and manage posts with type-safe user relations
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Create Post Form */}
            <Card className="bg-muted/50">
              <CardHeader>
                <CardTitle className="text-lg">Create New Post</CardTitle>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleCreatePost} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="postTitle">Title</Label>
                    <Input
                      id="postTitle"
                      type="text"
                      placeholder="Enter post title"
                      value={newPost.title}
                      onChange={(e) =>
                        setNewPost({ ...newPost, title: e.target.value })
                      }
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="postContent">Content</Label>
                    <Textarea
                      id="postContent"
                      placeholder="Write your post content..."
                      value={newPost.content}
                      onChange={(e) =>
                        setNewPost({ ...newPost, content: e.target.value })
                      }
                      rows={4}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="postAuthor">Author</Label>
                    <select
                      id="postAuthor"
                      value={newPost.user_id}
                      onChange={(e) =>
                        setNewPost({
                          ...newPost,
                          user_id: parseInt(e.target.value),
                        })
                      }
                      className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {usersQuery.data?.map((user) => (
                        <option key={user.id} value={user.id}>
                          {user.name} ({user.email})
                        </option>
                      ))}
                    </select>
                  </div>
                  <Button
                    type="submit"
                    disabled={createPostMutation.isPending}
                    variant="success"
                    className="w-full"
                  >
                    {createPostMutation.isPending
                      ? "Creating..."
                      : "Create Post"}
                  </Button>
                </form>
              </CardContent>
            </Card>

            {/* Posts List */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="font-semibold">All Posts</h3>
                <Badge variant="secondary">
                  {postsQuery.data?.length || 0} posts
                </Badge>
              </div>
              <div className="max-h-64 overflow-y-auto space-y-2">
                {postsQuery.data?.map((post) => (
                  <div
                    key={post.id}
                    className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50 transition-colors"
                  >
                    <div className="flex-1">
                      <div className="font-medium">{post.title}</div>
                      <div className="text-sm text-muted-foreground line-clamp-1">
                        {post.content}
                      </div>
                    </div>
                    <Button
                      onClick={() => handleGetPost(post.id)}
                      variant="outline"
                      size="sm"
                    >
                      View Details
                    </Button>
                  </div>
                ))}
              </div>
            </div>

            {/* Selected Post Details */}
            {selectedPost && (
              <Card className="bg-gradient-to-br from-accent-50 to-accent-100 border-accent-200">
                <CardHeader>
                  <CardTitle className="text-lg text-accent-900">
                    Selected Post Details
                    {getPostQuery.isPending && (
                      <Badge variant="secondary" className="ml-2">
                        Loading...
                      </Badge>
                    )}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="bg-white rounded-lg p-4 border border-accent-200">
                    <pre className="text-sm text-accent-800 overflow-x-auto">
                      {JSON.stringify(selectedPost, null, 2)}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
