import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { api } from "../lib/api-client";
import type { User } from "../types/generated";

export const Route = createFileRoute("/users")({
  component: UsersPage,
});

function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = async () => {
    setLoading(true);
    setError("");

    const result = await api.users.list();
    if (!result.success) {
      setError(result.error.detail);
    } else {
      setUsers(result.data || []);
    }
    setLoading(false);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="text-lg text-gray-600">Loading users...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold text-gray-900">Users</h1>

      {error && (
        <div className="p-4 border border-red-200 rounded-md bg-red-50">
          <div className="text-red-700">Error: {error}</div>
        </div>
      )}

      <div className="grid gap-4">
        {users.map((user) => (
          <div key={user.id} className="p-6 bg-white rounded-lg shadow">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-semibold text-gray-900">
                  {user.email}
                </h2>
                <p className="text-gray-600">{user.email}</p>
              </div>
              <div className="text-sm text-gray-500">ID: {user.id}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
