import {
  createFileRoute,
  isRedirect,
  useNavigate,
} from "@tanstack/react-router";
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
  IconChevronLeft,
  IconChevronRight,
} from "@tabler/icons-react";
import type { GitHubRepo } from "../../api/endpoints.schemas";

type WebhooksSearch = {
  page?: number;
};

export const Route = createFileRoute("/profile/webhooks")({
  validateSearch: (search: Record<string, unknown>): WebhooksSearch => ({
    page: Number(search.page) || 1,
  }),
  component: ProfileWebhooks,
});

const PER_PAGE = 10;

function ProfileWebhooks() {
  const { page = 1 } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });

  const {
    data: reposData,
    isLoading,
    isRefetching,
    error,
    refetch,
  } = useListReposApiGithubReposGet({ page, include_webhooks: true });

  const repos = reposData?.data?.repos ?? [];
  const webhookUrl = reposData?.data?.webhook_url ?? "";
  const hasNext = repos.length === PER_PAGE;

  const handlePageChange = (newPage: number) => {
    navigate({ search: { page: newPage } });
  };

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

      {isLoading ? (
        <p>Loading repositories...</p>
      ) : error ? (
        <p>Failed to load repositories. Please try again.</p>
      ) : (
        <>
          <ul className="repo-list">
            {repos.map((repo) => (
              <RepoItem
                key={repo.id}
                repo={repo}
                webhookUrl={webhookUrl}
                onUpdate={refetch}
                isRefetching={isRefetching}
              />
            ))}
          </ul>

          {repos.length === 0 && (
            <p className="empty-state">
              No repositories found with admin access.
            </p>
          )}

          {(page > 1 || hasNext) && (
            <div className="pagination">
              <button
                className="btn btn-small"
                onClick={() => handlePageChange(page - 1)}
                disabled={page <= 1}
              >
                <IconChevronLeft size={16} />
                Previous
              </button>
              <span className="pagination-current">Page {page}</span>
              <button
                className="btn btn-small"
                onClick={() => handlePageChange(page + 1)}
                disabled={!hasNext}
              >
                Next
                <IconChevronRight size={16} />
              </button>
            </div>
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
  isRefetching: boolean;
}

function RepoItem({ repo, webhookUrl, onUpdate, isRefetching }: RepoItemProps) {
  const createWebhook = useCreateWebhookApiGithubReposOwnerRepoWebhooksPost();
  const deleteWebhook =
    useDeleteWebhookApiGithubReposOwnerRepoWebhooksHookIdDelete();

  const existingHook = repo.webhooks?.find(
    (hook) => String(hook.config?.url ?? "") === webhookUrl,
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

  const isPending =
    createWebhook.isPending || deleteWebhook.isPending || isRefetching;

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
            <p className="description">{String(repo.description)}</p>
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
              {isPending ? "Working" : "Remove"}
            </button>
          ) : (
            <button
              className="btn btn-primary btn-small"
              onClick={handleAdd}
              disabled={isPending}
              title="Add webhook"
            >
              <IconPlus size={16} />
              {isPending ? "Working" : "Add"}
            </button>
          )}
        </div>
      </div>
    </li>
  );
}
