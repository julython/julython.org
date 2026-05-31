---
name: github-issues
description: Work with GitHub issues including listing, creating, updating, searching, labeling, and managing milestones. Use when the user mentions issues, milestones, labels, projects, or GitHub repository management.
---

# GitHub Issues

Work with GitHub issues using the `gh` CLI (preferred) or the GitHub REST API directly via bash.

## Setup

**Prerequisites:** Install the `gh` CLI:
```bash
brew install gh           # macOS
# or follow https://cli.github.com
gh auth login              # Authenticate with GitHub
```

**Required environment:**
```bash
export GITHUB_REPO="owner/repo"   # e.g., "julython/www"
```

## Listing Issues

```bash
# All open issues
gh issue list --repo $GITHUB_REPO --state open

# All closed issues
gh issue list --repo $GITHUB_REPO --state closed

# Recent activity (last 7 days)
gh issue list --repo $GITHUB_REPO --limit 50

# With full details (numbers, labels, assignees)
gh issue list --repo $GITHUB_REPO --json number,title,labels,assignees,state,comments,createdAt

# Filter by label
gh issue list --repo $GITHUB_REPO --label "bug"

# Filter by assignee
gh issue list --repo $GITHUB_REPO --assignee "username"

# Filter by milestone
gh issue list --repo $GITHUB_REPO --milestone "v1.0"

# List issues in a project (beta)
gh project list-issues --owner owner --number 1 --repo repo
```

## Creating Issues

```bash
# Simple issue
gh issue create --repo $GITHUB_REPO \
  --title "Feature request: dark mode" \
  --body "Users have requested dark mode support."

# With labels
gh issue create --repo $GITHUB_REPO \
  --title "Bug: login crash" \
  --body "App crashes on login when using SSO." \
  --label "bug,high-priority"

# With milestone
gh issue create --repo $GITHUB_REPO \
  --title "Refactor auth module" \
  --body "Migrate to OIDC." \
  --milestone "v2.0"

# From a template (if .github/ISSUE_TEMPLATE exists)
gh issue create --repo $GITHUB_REPO --template bug_report

# Link to a PR
gh issue create --repo $GITHUB_REPO \
  --title "Chore: update dependencies" \
  --body "See PR #123 for implementation." \
  --crossreference

# Assign users
gh issue create --repo $GITHUB_REPO \
  --title "Doc update needed" \
  --body "Readme is out of date." \
  --assignee "alice,bob"

# With project board
gh issue create --repo $GITHUB_REPO \
  --title "Fix payment flow" \
  --body "Payment fails for cards from country X." \
  --project "Project Name"
```

## Updating Issues

```bash
# Add labels
gh issue edit 42 --add-labels "bug,high-priority"

# Remove labels
gh issue edit 42 --remove-labels "stale"

# Add/assign someone
gh issue edit 42 --add-assignee "alice"

# Remove assignee
gh issue edit 42 --remove-assignee "bob"

# Add a milestone
gh issue edit 42 --milestone "v2.0"

# Remove milestone
gh issue edit 42 --milestone ""

# Update title
gh issue edit 42 --title "Updated: login crash fix"

# Update body
gh issue edit 42 --body "Additional context added."

# Link a PR (cross-reference)
gh issue edit 42 --add-close "owner/repo#50"

# Close/Reopen
gh issue close 42
gh issue reopen 42

# Add to project
gh project add-issue --owner owner --number 1 --id $(gh issue view 42 --json id -q .id)

# Move project item (reorder)
gh project update-item --owner owner --number 1 --id <projectItemId> --fields content:{order:"top"}
```

## Searching Issues

```bash
# Search across all repos you have access to
gh issue list --search "label:bug is:open sort:comments"

# Search closed issues in a milestone
gh issue list --repo $GITHUB_REPO --state closed --search "milestone:v2.0"

# Search by author
gh issue list --repo $GITHUB_REPO --search "author:alice"

# Search for issues without labels
gh issue list --repo $GITHUB_REPO --search "no:label"

# Combined filters (supports boolean logic)
gh issue list --repo $GITHUB_REPO --search "is:open label:bug -label:duplicate"
```

## Viewing Issue Details

```bash
# Full issue view
gh issue view 42

# JSON output for programmatic access
gh issue view 42 --json number,title,body,labels,assignees,state,comments,createdAt,closedAt

# Comments
gh issue comments 42

# Comment history with links to changed content
gh issue view 42 --comments
```

## Managing Labels

```bash
# List all labels
gh label list --repo $GITHUB_REPO

# Create labels (bulk)
gh label create "bug" --color "d73a4a" --description "Something isn't working" --repo $GITHUB_REPO
gh label create "enhancement" --color "a2eeef" --description "New feature" --repo $GITHUB_REPO
gh label create "documentation" --color "0075ca" --description "Documentation improvements" --repo $GITHUB_REPO
gh label create "good-first-issue" --color "7057ff" --description "Good for newcomers" --repo $GITHUB_REPO
gh label create "high-priority" --color "b60205" --description "Critical items" --repo $GITHUB_REPO
gh label create "stale" --color "ffffff" --description "Automatically marked stale" --repo $GITHUB_REPO
gh label create "duplicate" --color "cfd3d7" --description "Already exists" --repo $GITHUB_REPO

# Update labels (bulk)
gh label edit "bug" --new-name "bug-report" --new-color "d73a4a" --new-description "Fixed something" --repo $GITHUB_REPO

# Delete a label
gh label delete "old-label" --repo $GITHUB_REPO
```

## Milestones

```bash
# List milestones
gh milestone list --repo $GITHUB_REPO

# Create a milestone
gh milestone create --repo $GITHUB_REPO \
  --title "v2.0" \
  --description "Major release with OIDC support" \
  --due-date 2025-06-01

# Update a milestone (add/due date, state, description)
gh milestone edit 1 --repo $GITHUB_REPO --state closed --description "Updated description"

# List issues in a milestone
gh milestone view 1 --repo $GITHUB_REPO --json state,dueAt,issues
```

## Managing Issues (Bulk Operations)

```bash
# Mark stale issues (common pattern)
gh issue list --repo $GITHUB_REPO --label "stale" --state open --json number --jq '.[].number' | \
  xargs -I {} gh issue close {} --reason "not planned"

# Reopen stale issues that were reopened
gh issue list --repo $GITHUB_REPO --label "stale" --state closed --json number,title --jq '.[].number'

# Close issues older than N days without recent comments
gh issue list --repo $GITHUB_REPO --search "updated:<2025-01-01 comments:0" --state open

# Bulk label issues in a milestone
gh issue list --repo $GITHUB_REPO --milestone "v1.0" --json number | \
  jq -r '.[].number' | \
  xargs -I {} gh issue edit {} --add-labels "priority:low"
```

## Using the GitHub REST API (when gh is not available)

```bash
# List issues via REST API
curl -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/$GITHUB_REPO/issues?state=open&per_page=100"

# Create an issue
curl -X POST -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/$GITHUB_REPO/issues" \
  -d '{"title":"New issue","body":"Issue body","labels":["bug"]}'

# Update an issue
curl -X PATCH -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/$GITHUB_REPO/issues/42" \
  -d '{"state":"closed","labels":["bug","resolved"]}'

# Search issues (more powerful filters than gh)
curl -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/search/issues?q=repo:$GITHUB_REPO+is:issue+label:bug+state:open&sort=comments&order=desc"
```

**Token setup (REST API):**
```bash
export GITHUB_TOKEN=ghp_your_personal_access_token
# Generate at: https://github.com/settings/tokens (scope: repo)
```

## Useful Patterns

### Issue number from title (when you know the title)
```bash
gh issue list --repo $GITHUB_REPO --search "title:your title here" --json number,title
```

### Get issues by assignee across milestones
```bash
gh issue list --repo $GITHUB_REPO --assignee "alice" --json number,title,milestone,state
```

### Generate an issue summary report
```bash
gh issue list --repo $GITHUB_REPO --state all --json number,title,state,labels,assignees,comments,createdAt | \
  jq '[group_by(.state)[] | {state: .[0].state, count: length, first: .[0].number}]'
```

### Link related issues (duplicate/close references)
```bash
gh issue edit 42 --body "Duplicate of #100. See also #205."
gh issue close 43 --close-as-duplicate 100
```

### Reference GitHub docs
- `gh issue --help` — full CLI reference
- [GitHub Issues docs](https://docs.github.com/en/issues/tracking-your-work-with-issues)
- [GitHub REST API: Issues](https://docs.github.com/en/rest/issues)
