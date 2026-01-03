import { useState } from "react";
import { IconGitCommit, IconFolder } from "@tabler/icons-react";
import type { Leader } from "../api/endpoints.schemas";

interface UserStatsProps {
  profile: Leader;
}

type TabId = "projects" | "commits";

export function UserStats({ profile }: UserStatsProps) {
  const [activeTab, setActiveTab] = useState<TabId>("projects");

  return (
    <div className="user-stats">
      <nav className="profile-nav">
        <ul>
          <li>
            <a
              href="#projects"
              className={activeTab === "projects" ? "active" : ""}
              onClick={(e) => {
                e.preventDefault();
                setActiveTab("projects");
              }}
            >
              <IconFolder size={18} /> Projects
            </a>
          </li>
          <li>
            <a
              href="#commits"
              className={activeTab === "commits" ? "active" : ""}
              onClick={(e) => {
                e.preventDefault();
                setActiveTab("commits");
              }}
            >
              <IconGitCommit size={18} /> Commits
            </a>
          </li>
        </ul>
      </nav>

      <div className="tab-content">
        {activeTab === "projects" && <ProjectsTab profile={profile} />}
        {activeTab === "commits" && <CommitsTab profile={profile} />}
      </div>
    </div>
  );
}

function ProjectsTab({ profile }: { profile: Leader }) {
  const projects: any[] = profile.project_count ? [] : [];

  if (projects.length === 0) {
    return (
      <div className="empty-state">
        <p>No projects yet.</p>
      </div>
    );
  }

  return (
    <ul className="repo-list">
      {projects.map((project) => (
        <li key={project.id} className="repo-item">
          <div className="repo-main">
            <div className="repo-info">
              <h4>
                <a href={project.url} target="_blank" rel="noopener noreferrer">
                  {project.name}
                </a>
              </h4>
              {project.description && (
                <p className="description">{project.description}</p>
              )}
            </div>
            <span className="badge badge-info">
              {project.commit_count ?? 0} commits
            </span>
          </div>
        </li>
      ))}
    </ul>
  );
}

function CommitsTab({ profile }: { profile: Leader }) {
  const commits: any[] = profile.commit_count ? [] : [];

  if (commits.length === 0) {
    return (
      <div className="empty-state">
        <p>No commits yet.</p>
      </div>
    );
  }

  return (
    <ul className="commits-list">
      {commits.map((commit) => (
        <li key={commit.id} className="commit-item">
          <div className="commit-header">
            <a
              href={commit.url}
              target="_blank"
              rel="noopener noreferrer"
              className="commit-hash"
            >
              {commit.hash?.slice(0, 7)}
            </a>
            <span className="commit-date">
              {commit.timestamp &&
                new Date(commit.timestamp).toLocaleDateString()}
            </span>
          </div>
          <p className="commit-message">{commit.message}</p>
          {commit.project && (
            <span className="commit-project">in {commit.project.name}</span>
          )}
        </li>
      ))}
    </ul>
  );
}
