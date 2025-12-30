import { createRootRoute, Link, Outlet } from "@tanstack/react-router";
import { useEffect } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AccountIcon } from "../components/AccountIcon";

const queryClient = new QueryClient();

function RootLayout() {
  return (
    <QueryClientProvider client={queryClient}>
      <Header />
      <Outlet />
    </QueryClientProvider>
  );
}

function Header() {
  useEffect(() => {
    const saved = localStorage.getItem("theme") as "light" | "dark" | null;
    const prefersDark = window.matchMedia(
      "(prefers-color-scheme: dark)",
    ).matches;
    const initial = saved || (prefersDark ? "dark" : "light");
    document.documentElement.setAttribute("data-theme", initial);
  }, []);

  return (
    <nav className="navbar" id="topnav">
      <div className="navbar-inner">
        <Link to="/" className="brand">
          <div className="logo" />
        </Link>
        <ul className="nav">
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
        </ul>
        <AccountIcon />
      </div>
    </nav>
  );
}

export const Route = createRootRoute({
  component: RootLayout,
});
