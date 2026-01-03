import { Link } from "@tanstack/react-router";
import { IconBrandGithub } from "@tabler/icons-react";
import type { Leader } from "../api/endpoints.schemas";

interface UserProfileCardProps {
  profile: Leader;
  isOwnProfile?: boolean;
}

export function UserProfileCard({
  profile,
  isOwnProfile,
}: UserProfileCardProps) {
  const displayName = profile.name || profile.username;
  const points = profile.points ?? 0;

  return (
    <div className="profile-card">
      <header>
        {profile.avatar_url ? (
          <img className="avatar" src={profile.avatar_url} alt={displayName} />
        ) : (
          <img
            className="avatar"
            src="/images/spread_the_word_button.png"
            alt={displayName}
          />
        )}

        <div>
          <h3>{displayName}</h3>
          {profile.name && profile.username && (
            <p className="username">@{profile.username}</p>
          )}
          <span className="points">{points} points</span>
        </div>
      </header>

      {profile.name && <p className="subtitle">{profile.name}</p>}

      <div className="profile-links">
        {profile.username && (
          <a
            href={`https://github.com/${profile.username}`}
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-small"
          >
            <IconBrandGithub size={16} /> GitHub
          </a>
        )}
      </div>

      {isOwnProfile && (
        <div className="profile-actions">
          <Link to="/profile/edit" className="btn btn-primary">
            Edit Profile
          </Link>
        </div>
      )}
    </div>
  );
}
