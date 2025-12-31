import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { useGetUserSessionAuthSessionGet } from "../../api/auth/auth";

export const Route = createFileRoute("/profile/edit")({
  component: ProfileEdit,
});

function ProfileEdit() {
  const { data: session, isLoading } = useGetUserSessionAuthSessionGet();
  const user = session?.data?.user;

  const [name, setName] = useState("");
  const [location, setLocation] = useState("");
  const [team, setTeam] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (user) {
      setName(user.name ?? "");
      setLocation("tbd");
      setTeam("tbd");
    }
  }, [user]);

  if (isLoading) {
    return <p>Loading...</p>;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    // TODO: Call profile update API
    await new Promise((r) => setTimeout(r, 500));
    setIsSaving(false);
  };

  return (
    <div>
      <h2 className="what-is-this">Edit Profile</h2>

      <form onSubmit={handleSubmit}>
        <div className="form-section">
          <h3>Basic Information</h3>

          <label>
            Display Name
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Your name"
            />
          </label>

          <label>
            Location
            <input
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              placeholder="City, Country"
            />
            <small>Used for location-based leaderboards</small>
          </label>

          <label>
            Team
            <input
              type="text"
              value={team}
              onChange={(e) => setTeam(e.target.value)}
              placeholder="Your team name"
            />
            <small>Join or create a team to compete together</small>
          </label>
        </div>

        <div className="form-section">
          <h3>Email Addresses</h3>
          <p className="lead">
            Add all email addresses you use for git commits. We use these to
            match commits to your account.
          </p>
          <EmailList />
        </div>

        <div className="form-section">
          <h3>Connected Accounts</h3>
          <ConnectedAccounts />
        </div>

        <button type="submit" className="btn" disabled={isSaving}>
          {isSaving ? "Saving..." : "Save Changes"}
        </button>
      </form>
    </div>
  );
}

function EmailList() {
  const [emails, setEmails] = useState<string[]>(["example@gmail.com"]);
  const [newEmail, setNewEmail] = useState("");

  const handleAdd = () => {
    if (newEmail && !emails.includes(newEmail)) {
      setEmails([...emails, newEmail]);
      setNewEmail("");
    }
  };

  const handleRemove = (email: string) => {
    setEmails(emails.filter((e) => e !== email));
  };

  return (
    <div>
      <ul className="repo-list">
        {emails.map((email) => (
          <li key={email} className="repo-item">
            <div className="repo-main">
              <span>{email}</span>
              <button
                type="button"
                className="btn-link"
                onClick={() => handleRemove(email)}
              >
                Remove
              </button>
            </div>
          </li>
        ))}
      </ul>

      <div className="toolbar">
        <input
          type="email"
          value={newEmail}
          onChange={(e) => setNewEmail(e.target.value)}
          placeholder="Add email address"
          onKeyDown={(e) =>
            e.key === "Enter" && (e.preventDefault(), handleAdd())
          }
        />
        <button type="button" className="btn" onClick={handleAdd}>
          Add
        </button>
      </div>
    </div>
  );
}

function ConnectedAccounts() {
  const accounts = [
    { provider: "github", username: "user", connected: true },
    { provider: "gitlab", username: null, connected: false },
  ];

  return (
    <ul className="repo-list">
      {accounts.map((account) => (
        <li
          key={account.provider}
          className={`repo-item ${account.connected ? "has-webhook" : ""}`}
        >
          <div className="repo-main">
            <div>
              <strong style={{ textTransform: "capitalize" }}>
                {account.provider}
              </strong>
              {account.connected && (
                <span
                  style={{
                    marginLeft: "0.5rem",
                    color: "var(--text-secondary)",
                  }}
                >
                  @{account.username}
                </span>
              )}
            </div>
            {account.connected ? (
              <span className="status active">Connected</span>
            ) : (
              <a
                href={`/auth/login/${account.provider}`}
                className="btn btn-small"
              >
                Connect
              </a>
            )}
          </div>
        </li>
      ))}
    </ul>
  );
}
