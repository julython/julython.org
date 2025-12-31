import {
  createFileRoute,
  Link,
  Outlet,
  redirect,
} from "@tanstack/react-router";
import { getUserSessionAuthSessionGet } from "../api/auth/auth";
import { IconUser, IconWebhook, IconSettings } from "@tabler/icons-react";

export const Route = createFileRoute("/profile")({
  beforeLoad: async () => {
    try {
      await getUserSessionAuthSessionGet();
    } catch {
      throw redirect({ to: "/", search: { error: "unauthenticated" } });
    }
  },
  component: ProfileLayout,
});

function ProfileLayout() {
  return (
    <div className="profile-layout">
      <aside className="profile-sidebar">
        <ProfileNav />
      </aside>
      <main className="profile-main">
        <Outlet />
      </main>
    </div>
  );
}

function ProfileNav() {
  return (
    <nav className="profile-nav">
      <h4>Profile</h4>
      <ul className="nav-list">
        <li>
          <Link
            to="/profile"
            activeOptions={{ exact: true }}
            activeProps={{ className: "active" }}
          >
            <IconUser size={18} />
            <span>Overview</span>
          </Link>
        </li>
        <li>
          <Link to="/profile/webhooks" activeProps={{ className: "active" }}>
            <IconWebhook size={18} />
            <span>Webhooks</span>
          </Link>
        </li>
        <li>
          <Link to="/profile/edit" activeProps={{ className: "active" }}>
            <IconSettings size={18} />
            <span>Settings</span>
          </Link>
        </li>
      </ul>
    </nav>
  );
}
