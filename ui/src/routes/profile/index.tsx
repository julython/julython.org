import { createFileRoute, Link } from "@tanstack/react-router";
import { useGetUserSessionAuthSessionGet } from "../../api/auth/auth";

export const Route = createFileRoute("/profile/")({
  component: ProfileIndex,
});

function ProfileIndex() {
  const { data: session, isLoading } = useGetUserSessionAuthSessionGet();

  if (isLoading) {
    return <p>Loading...</p>;
  }

  const user = session?.data?.user;

  if (!user) {
    return <p>Unable to load profile.</p>;
  }

  return (
    <div className="profile-overview">
      <h2 className="what-is-this">Profile</h2>

      <div className="profile-card">
        <header>
          {user.avatar_url && (
            <img
              src={user.avatar_url}
              alt={user.name || user.username || "anon"}
              className="avatar"
            />
          )}
          <div>
            <h3>{user.name || user.username}</h3>
            {user.name && <p className="username">@{user.username}</p>}
          </div>
        </header>

        <Link to="/profile/edit" className="btn">
          Edit Profile
        </Link>
      </div>
    </div>
  );
}
