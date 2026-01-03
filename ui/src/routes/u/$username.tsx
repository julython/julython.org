import { createFileRoute, Link } from "@tanstack/react-router";
import { useGetLeaderApiV1GameLeadersUsernameGet } from "../../api/game/game";
import { useGetUserSessionAuthSessionGet } from "../../api/auth/auth";
import { UserProfileCard } from "../../components/UserProfileCard";
import { UserStats } from "../../components/UserStats";

export const Route = createFileRoute("/u/$username")({
  component: UserProfile,
});

function UserProfile() {
  const { username } = Route.useParams();
  const { data: session } = useGetUserSessionAuthSessionGet();
  const {
    data: leader,
    isLoading,
    error,
  } = useGetLeaderApiV1GameLeadersUsernameGet(username);

  const isOwnProfile = session?.data?.user?.username === username;

  if (isLoading) {
    return (
      <div className="container section-container no-border">
        <div className="row">
          <div className="span10 offset1">
            <p>Loading profile...</p>
          </div>
        </div>
      </div>
    );
  }

  if (error || !leader?.data) {
    return (
      <div className="container section-container no-border">
        <div className="row">
          <div className="span10 offset1">
            <h2 className="what-is-this">User Not Found</h2>
            <p className="subtitle">
              The user "{username}" could not be found.
            </p>
            <Link to="/" className="btn btn-primary">
              Back to Home
            </Link>
          </div>
        </div>
      </div>
    );
  }

  const profile = leader.data;

  return (
    <div className="container section-container no-border">
      <div className="row">
        <div className="span10 offset1">
          <UserProfileCard profile={profile} isOwnProfile={isOwnProfile} />
          <UserStats profile={profile} />
        </div>
      </div>
    </div>
  );
}
