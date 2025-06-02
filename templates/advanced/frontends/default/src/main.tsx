import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "@tanstack/react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { createRouter } from "./router";
import "./styles.css";
import reportWebVitals from "./reportWebVitals";
import { auth } from "./lib/api-client";

// Create a QueryClient instance
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000, // 10 minutes
      retry: 1,
    },
  },
});

auth.setToken("demo-token");

// Create a new router instance
const router = createRouter();

// Render the app
const rootElement = document.getElementById("app");
if (rootElement) {
  // Check if we have server-rendered content (SSR) or need to render from scratch (SPA)
  const hasServerRenderedContent = rootElement.innerHTML.trim() !== "";

  if (hasServerRenderedContent) {
    // Hydrate the server-rendered content
    ReactDOM.hydrateRoot(
      rootElement,
      <StrictMode>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </StrictMode>
    );
  } else {
    // Render from scratch (SPA mode)
    const root = ReactDOM.createRoot(rootElement);
    root.render(
      <StrictMode>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </StrictMode>
    );
  }
}

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
