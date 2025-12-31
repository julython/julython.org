import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import {
  useListReposApiGithubReposGet,
  useCreateWebhookApiGithubReposOwnerRepoWebhooksPost,
  useDeleteWebhookApiGithubReposOwnerRepoWebhooksHookIdDelete,
} from "../../api/git-hub/git-hub";
import {
  IconWebhook,
  IconPlus,
  IconTrash,
  IconLock,
} from "@tabler/icons-react";
import type { GitHubRepo } from "../../api/endpoints.schemas";

export const Route = createFileRoute("/profile/webhooks")({
  component: ProfileWebhooks,
});

function ProfileWebhooks() {
  const [filter, setFilter] = useState("");

  const {
    data: reposData,
    isLoading,
    isRefetching,
    error,
    refetch,
  } = useListReposApiGithubReposGet({ include_webhooks: true });

  const repos = reposData?.data?.repos ?? [];
  const webhookUrl = reposData?.data?.webhook_url ?? "";

  const filteredRepos = repos.filter(
    (repo) =>
      repo.full_name.toLowerCase().includes(filter.toLowerCase()) ||
      repo.description?.toLowerCase().includes(filter.toLowerCase()),
  );

  if (error?.response?.status === 401) {
    return (
      <div>
        <h2 className="what-is-this">Webhooks</h2>
        <p>You need to connect your GitHub account to manage webhooks.</p>
        <a href="/auth/login/github" className="btn">
          Connect GitHub
        </a>
      </div>
    );
  }

  return (
    <div>
      <h2 className="what-is-this">Webhooks</h2>
      <p className="subtitle">
        Add webhooks to your repositories to track commits during Julython.
      </p>

      {isLoading || isRefetching ? (
        <p>Loading repositories...</p>
      ) : error ? (
        <p>Failed to load repositories. Please try again.</p>
      ) : (
        <>
          <div className="toolbar">
            <input
              type="search"
              placeholder="Filter repositories..."
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
            />
            <span>{filteredRepos.length} repositories</span>
          </div>

          <ul className="repo-list">
            {filteredRepos.map((repo) => (
              <RepoItem
                key={repo.id}
                repo={repo}
                webhookUrl={webhookUrl}
                onUpdate={() => refetch()}
              />
            ))}
          </ul>

          {filteredRepos.length === 0 && (
            <p className="empty-state">
              {filter
                ? "No repositories match your filter."
                : "No repositories found with admin access."}
            </p>
          )}
        </>
      )}
    </div>
  );
}

interface RepoItemProps {
  repo: GitHubRepo;
  webhookUrl: string;
  onUpdate: () => void;
}

function RepoItem({ repo, webhookUrl, onUpdate }: RepoItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const createWebhook = useCreateWebhookApiGithubReposOwnerRepoWebhooksPost();
  const deleteWebhook =
    useDeleteWebhookApiGithubReposOwnerRepoWebhooksHookIdDelete();

  const existingHook = repo.webhooks?.find(
    (hook) => hook.config?.url === webhookUrl,
  );
  const hasWebhook = !!existingHook;

  const handleAdd = async () => {
    try {
      await createWebhook.mutateAsync({ owner: repo.owner, repo: repo.name });
      onUpdate();
    } catch (err) {
      console.error("Failed to create webhook:", err);
    }
  };

  const handleRemove = async () => {
    if (!existingHook) return;
    try {
      await deleteWebhook.mutateAsync({
        owner: repo.owner,
        repo: repo.name,
        hookId: existingHook.id,
      });
      onUpdate();
    } catch (err) {
      console.error("Failed to delete webhook:", err);
    }
  };

  const isPending = createWebhook.isPending || deleteWebhook.isPending;

  return (
    <li className={`repo-item ${hasWebhook ? "has-webhook" : ""}`}>
      <div className="repo-main">
        <div className="repo-info">
          <h4>
            <a href={repo.html_url} target="_blank" rel="noopener noreferrer">
              {repo.full_name}
            </a>
            {repo.private && (
              <span className="badge private" title="Private repository">
                <IconLock size={12} />
              </span>
            )}
            {hasWebhook && (
              <span className="badge webhook" title="Webhook active">
                <IconWebhook size={12} />
              </span>
            )}
          </h4>
          {repo.description && (
            <p className="description">{repo.description}</p>
          )}
        </div>

        <div className="repo-actions">
          {hasWebhook ? (
            <button
              className="btn btn-danger btn-small"
              onClick={handleRemove}
              disabled={isPending}
              title="Remove webhook"
            >
              <IconTrash size={16} />
              {isPending ? "Removing..." : "Remove"}
            </button>
          ) : (
            <button
              className="btn btn-primary btn-small"
              onClick={handleAdd}
              disabled={isPending}
              title="Add webhook"
            >
              <IconPlus size={16} />
              {isPending ? "Adding..." : "Add"}
            </button>
          )}
        </div>
      </div>

      {repo.webhooks && repo.webhooks.length > 0 && (
        <>
          <button
            className="btn-link expand-toggle"
            onClick={() => setIsExpanded(!isExpanded)}
          >
            {isExpanded ? "Hide" : "Show"} {repo.webhooks.length} webhook
            {repo.webhooks.length !== 1 && "s"}
          </button>

          {isExpanded && (
            <ul className="webhook-list">
              {repo.webhooks.map((hook) => (
                <li
                  key={hook.id}
                  className={hook.config?.url === webhookUrl ? "ours" : ""}
                >
                  <span className="webhook-url">
                    {String(hook.config?.url ?? "Unknown URL")}
                  </span>
                  <span
                    className={`status ${hook.active ? "active" : "inactive"}`}
                  >
                    {hook.active ? " Active" : " Inactive"}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </li>
  );
}
