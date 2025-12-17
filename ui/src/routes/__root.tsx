import { createRootRoute, Link, Outlet } from "@tanstack/solid-router";
import { createSignal, onMount } from "solid-js";

function RootLayout() {
  const [theme, setTheme] = createSignal<"light" | "dark">("light");

  onMount(() => {
    const saved = localStorage.getItem("theme") as "light" | "dark" | null;
    const prefersDark = window.matchMedia(
      "(prefers-color-scheme: dark)",
    ).matches;
    const initial = saved || (prefersDark ? "dark" : "light");
    setTheme(initial);
    document.documentElement.setAttribute("data-theme", initial);
  });

  const toggleTheme = () => {
    const next = theme() === "light" ? "dark" : "light";
    setTheme(next);
    document.documentElement.setAttribute("data-theme", next);
    localStorage.setItem("theme", next);
  };

  return (
    <>
      <nav class="navbar" id="topnav">
        <div class="navbar-inner">
          <Link to="/" class="brand">
            <div class="logo" />
          </Link>
          <ul class="nav">
            <li>
              <Link to="/">Leaders</Link>
            </li>
            <li>
              <Link to="/">Projects</Link>
            </li>
            <li>
              <Link to="/">Blog</Link>
            </li>
            <li>
              <Link to="/help">Help</Link>
            </li>
            <li>
              <a href="/auth/login/github">Sign In</a>
            </li>
          </ul>
          <button class="theme-toggle" onClick={toggleTheme}>
            {theme() === "light" ? "🌙" : "☀️"}
          </button>
        </div>
      </nav>
      <Outlet />
    </>
  );
}

export const Route = createRootRoute({
  component: RootLayout,
});
