import { createFileRoute } from "@tanstack/react-router";
import { api } from "../lib/api-client";
import type { User } from "../types/generated";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Badge,
  Alert,
  AlertDescription,
} from "@/components/ui";
import { formatRelativeTime } from "@/lib/utils";

export const Route = createFileRoute("/users")({
  component: UsersPage,
});

function UsersPage() {
  const usersQuery = api.users.list.useQuery();

  if (usersQuery.isLoading) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="text-lg text-muted-foreground">Loading users...</div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Users</h1>
        <p className="text-muted-foreground mt-2">
          Manage system users and their information
        </p>
      </div>

      {/* Error Display */}
      {usersQuery.error && (
        <Alert variant="destructive">
          <AlertDescription>
            <strong>Error:</strong> {usersQuery.error.message}
          </AlertDescription>
        </Alert>
      )}

      {/* Users Grid */}
      {usersQuery.data && usersQuery.data.users.length > 0 ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {usersQuery.data.users.map((user: User) => (
            <Card key={user.id} className="hover:shadow-lg transition-shadow">
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <CardTitle className="text-lg">
                      {user.name || user.email}
                    </CardTitle>
                    <p className="text-muted-foreground mt-1">{user.email}</p>
                    <div className="flex items-center gap-2 mt-3">
                      <Badge variant="secondary">ID: {user.id}</Badge>
                      {user.age && (
                        <Badge variant="outline">Age: {user.age}</Badge>
                      )}
                    </div>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="text-sm text-muted-foreground space-y-1">
                  {user.createdAt && (
                    <p>Created: {formatRelativeTime(user.createdAt)}</p>
                  )}
                  {user.updatedAt && (
                    <p>Updated: {formatRelativeTime(user.updatedAt)}</p>
                  )}
                </div>
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
                  d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z"
                />
              </svg>
            </div>
            <h3 className="text-lg font-semibold mb-2">No users yet</h3>
            <p className="text-muted-foreground">
              No users have been created yet.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
