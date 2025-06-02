import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "@tanstack/react-router";

import { createRouter } from "./router";
import "./styles.css";
import reportWebVitals from "./reportWebVitals.ts";
import { auth } from "./lib/api-client.ts";

// Create a new router instance
const router = createRouter();
auth.setToken("demo-token");

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
        <RouterProvider router={router} />
      </StrictMode>
    );
  } else {
    // Render from scratch (SPA mode)
    const root = ReactDOM.createRoot(rootElement);
    root.render(
      <StrictMode>
        <RouterProvider router={router} />
      </StrictMode>
    );
  }
}

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
