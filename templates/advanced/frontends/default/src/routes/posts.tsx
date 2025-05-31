import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api-client";
import type { User, Post } from "../types/generated";
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
import { formatRelativeTime } from "@/lib/utils";

export const Route = createFileRoute("/posts")({
  component: PostsPage,
});

function PostsPage() {
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [newPost, setNewPost] = useState({
    title: "",
    content: "",
    author_id: 1,
  });

  // Query for posts
  const postsQuery = api.posts.list.useQuery();

  // Query for users
  const usersQuery = api.users.list.useQuery();

  // Mutation for creating posts
  const createPostMutation = api.posts.create.useMutation({
    onSuccess: () => {
      // Invalidate and refetch posts
      queryClient.invalidateQueries({ queryKey: ["posts"] });
      setNewPost({ title: "", content: "", author_id: 1 });
      setShowForm(false);
    },
  });

  const handleCreatePost = async (e: React.FormEvent) => {
    e.preventDefault();
    createPostMutation.mutate({
      title: newPost.title,
      content: newPost.content,
      user_id: newPost.author_id,
      published: false,
    });
  };

  const getUserName = (authorId: number) => {
    if (!usersQuery.data) return "Unknown";
    const user = usersQuery.data.find((u: User) => u.id === authorId);
    return user ? user.name : "Unknown";
  };

  const isLoading = postsQuery.isLoading || usersQuery.isLoading;
  const error = postsQuery.error || usersQuery.error;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="text-lg text-muted-foreground">Loading posts...</div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Posts</h1>
          <p className="text-muted-foreground mt-2">
            Manage your blog posts with full type safety
          </p>
        </div>
        <Button
          onClick={() => setShowForm(!showForm)}
          variant={showForm ? "secondary" : "brand"}
        >
          {showForm ? "Cancel" : "New Post"}
        </Button>
      </div>

      {/* Error Display */}
      {(error || createPostMutation.error) && (
        <Alert variant="destructive">
          <AlertDescription>
            <strong>Error:</strong>{" "}
            {error?.message || createPostMutation.error?.message}
          </AlertDescription>
        </Alert>
      )}

      {/* Create Post Form */}
      {showForm && (
        <Card>
          <CardHeader>
            <CardTitle>Create New Post</CardTitle>
            <CardDescription>
              Write a new blog post for your audience
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCreatePost} className="space-y-6">
              <div className="space-y-2">
                <Label htmlFor="title">Title</Label>
                <Input
                  id="title"
                  type="text"
                  value={newPost.title}
                  onChange={(e) =>
                    setNewPost({ ...newPost, title: e.target.value })
                  }
                  placeholder="Enter a compelling title..."
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="content">Content</Label>
                <Textarea
                  id="content"
                  value={newPost.content}
                  onChange={(e) =>
                    setNewPost({ ...newPost, content: e.target.value })
                  }
                  placeholder="Write your post content here..."
                  rows={6}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="author">Author</Label>
                <select
                  id="author"
                  value={newPost.author_id}
                  onChange={(e) =>
                    setNewPost({
                      ...newPost,
                      author_id: parseInt(e.target.value),
                    })
                  }
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {usersQuery.data?.map((user: User) => (
                    <option key={user.id} value={user.id}>
                      {user.name} ({user.email})
                    </option>
                  ))}
                </select>
              </div>

              <div className="flex gap-3 pt-4">
                <Button
                  type="submit"
                  disabled={createPostMutation.isPending}
                  variant="success"
                >
                  {createPostMutation.isPending ? "Creating..." : "Create Post"}
                </Button>
                <Button
                  type="button"
                  onClick={() => setShowForm(false)}
                  variant="outline"
                >
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {/* Posts Grid */}
      {postsQuery.data && postsQuery.data.length > 0 ? (
        <div className="grid gap-6">
          {postsQuery.data.map((post: Post) => (
            <Card key={post.id} className="hover:shadow-lg transition-shadow">
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <CardTitle className="text-xl mb-2">{post.title}</CardTitle>
                    <div className="flex items-center gap-4 text-sm text-muted-foreground">
                      <span>By {getUserName(post.user_id)}</span>
                      <Badge variant={post.published ? "success" : "warning"}>
                        {post.published ? "Published" : "Draft"}
                      </Badge>
                      {post.created_at && (
                        <span>{formatRelativeTime(post.created_at)}</span>
                      )}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button variant="ghost" size="sm">
                      Edit
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-muted-foreground leading-relaxed">
                  {post.content}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="text-center py-12">
            <div className="w-12 h-12 bg-muted rounded-lg flex items-center justify-center mx-auto mb-4">
              <svg
                className="w-6 h-6 text-muted-foreground"
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
            <h3 className="text-lg font-semibold mb-2">No posts yet</h3>
            <p className="text-muted-foreground mb-4">
              Create your first post to get started sharing your thoughts!
            </p>
            <Button onClick={() => setShowForm(true)} variant="brand">
              Create Your First Post
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
