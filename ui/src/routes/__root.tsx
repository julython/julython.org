import { createRootRoute, Link, Outlet } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AccountIcon } from "../components/AccountIcon";
import { IconSun, IconMoon } from "@tabler/icons-react";

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
  const [theme, setTheme] = useState<"light" | "dark">("light");

  useEffect(() => {
    const saved = localStorage.getItem("theme") as "light" | "dark" | null;
    const prefersDark = window.matchMedia(
      "(prefers-color-scheme: dark)",
    ).matches;
    const initial = saved || (prefersDark ? "dark" : "light");
    setTheme(initial);
    document.documentElement.setAttribute("data-theme", initial);
  }, []);

  const toggleTheme = () => {
    const next = theme === "light" ? "dark" : "light";
    setTheme(next);
    document.documentElement.setAttribute("data-theme", next);
    localStorage.setItem("theme", next);
  };

  return (
    <nav className="navbar" id="topnav">
      <div className="navbar-inner">
        <Link to="/" className="brand">
          <div className="logo" />
        </Link>
        <ul className="nav">
          <li>
            <Link to="/leaders">Leaders</Link>
          </li>
          <li>
            <Link to="/projects">Projects</Link>
          </li>
          <li>
            <Link to="/">Blog</Link>
          </li>
          <li>
            <Link to="/help">Help</Link>
          </li>
        </ul>
        <button className="theme-toggle" onClick={toggleTheme}>
          {theme === "light" ? <IconMoon size={20} /> : <IconSun size={20} />}
        </button>
        <AccountIcon />
      </div>
    </nav>
  );
}

export const Route = createRootRoute({
  component: RootLayout,
});
