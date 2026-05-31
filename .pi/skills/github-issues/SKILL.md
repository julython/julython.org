---
name: github-issues
description: Work with GitHub issues including listing, creating, updating, searching, labeling, managing milestones, and working through milestone task lists. Use when the user mentions issues, milestones, labels, projects, working on a plan, or GitHub repository management.
---

# GitHub Issues

Work with GitHub issues using the `gh` CLI (preferred) or the GitHub REST API directly via bash.

## Listing Issues

```bash
# All open issues
gh issue list --repo julython/julython.org --state open

# All closed issues
gh issue list --repo julython/julython.org --state closed

# Recent activity (last 7 days)
gh issue list --repo julython/julython.org --limit 50

# With full details (numbers, labels, assignees)
gh issue list --repo julython/julython.org --json number,title,labels,assignees,state,comments,createdAt

# Filter by label
gh issue list --repo julython/julython.org --label "bug"

# Filter by assignee
gh issue list --repo julython/julython.org --assignee "username"

# Filter by milestone
gh issue list --repo julython/julython.org --milestone "v1.0"

# List issues in a project (beta)
gh project list-issues --owner owner --number 1 --repo repo
```

## Creating Issues

```bash
# Simple issue
gh issue create --repo julython/julython.org \
  --title "Feature request: dark mode" \
  --body "Users have requested dark mode support."

# With labels
gh issue create --repo julython/julython.org \
  --title "Bug: login crash" \
  --body "App crashes on login when using SSO." \
  --label "bug,high-priority"

# With milestone
gh issue create --repo julython/julython.org \
  --title "Refactor auth module" \
  --body "Migrate to OIDC." \
  --milestone "v2.0"

# From a template (if .github/ISSUE_TEMPLATE exists)
gh issue create --repo julython/julython.org --template bug_report

# Link to a PR
gh issue create --repo julython/julython.org \
  --title "Chore: update dependencies" \
  --body "See PR #123 for implementation." \
  --crossreference

# Assign users
gh issue create --repo julython/julython.org \
  --title "Doc update needed" \
  --body "Readme is out of date." \
  --assignee "alice,bob"

# With project board
gh issue create --repo julython/julython.org \
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
gh issue list --repo julython/julython.org --state closed --search "milestone:v2.0"

# Search by author
gh issue list --repo julython/julython.org --search "author:alice"

# Search for issues without labels
gh issue list --repo julython/julython.org --search "no:label"

# Combined filters (supports boolean logic)
gh issue list --repo julython/julython.org --search "is:open label:bug -label:duplicate"
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
gh label list --repo julython/julython.org

# Create labels (bulk)
gh label create "bug" --color "d73a4a" --description "Something isn't working" --repo julython/julython.org
gh label create "enhancement" --color "a2eeef" --description "New feature" --repo julython/julython.org
gh label create "documentation" --color "0075ca" --description "Documentation improvements" --repo julython/julython.org
gh label create "good-first-issue" --color "7057ff" --description "Good for newcomers" --repo julython/julython.org
gh label create "high-priority" --color "b60205" --description "Critical items" --repo julython/julython.org
gh label create "stale" --color "ffffff" --description "Automatically marked stale" --repo julython/julython.org
gh label create "duplicate" --color "cfd3d7" --description "Already exists" --repo julython/julython.org

# Update labels (bulk)
gh label edit "bug" --new-name "bug-report" --new-color "d73a4a" --new-description "Fixed something" --repo julython/julython.org

# Delete a label
gh label delete "old-label" --repo julython/julython.org
```

## Milestones

```bash
# List milestones
gh api repos/julython/julython.org/milestones --jq '.[] | {number, title, state}'

# List open milestones (for milestone selection)
gh api repos/julython/julython.org/milestones --state open --jq '.[] | {number, title}'

# Create a milestone (via REST API)
gh api repos/julython/julython.org/milestones -X POST -d '{"title":"v2.0","state":"open"}'

# Create milestone with description
gh api repos/julython/julython.org/milestones -X POST -d '{"title":"v2.0","state":"open","description":"Major release with OIDC support","due_on":"2025-06-01"}'

# Update milestone description (the milestone becomes the epic — no parent issue needed)
gh api repos/julython/julython.org/milestones/2 -X PATCH -f "title=player-boards" -f "description=Personalized player board page..."

# List issues in a milestone
gh issue list --repo julython/julython.org --milestone "v2.0" --json number,title

# Check milestone exists (returns 0 if exists, 1 if not)
gh api repos/julython/julython.org/milestones --jq '.[] | select(.title=="v2.0") | .number'
```

## Managing Issues (Bulk Operations)

```bash
# Mark stale issues (common pattern)
gh issue list --repo julython/julython.org --label "stale" --state open --json number --jq '.[].number' | \
  xargs -I {} gh issue close {} --reason "not planned"

# Reopen stale issues that were reopened
gh issue list --repo julython/julython.org --label "stale" --state closed --json number,title --jq '.[].number'

# Close issues older than N days without recent comments
gh issue list --repo julython/julython.org --search "updated:<2025-01-01 comments:0" --state open

# Bulk label issues in a milestone
gh issue list --repo julython/julython.org --milestone "v1.0" --json number | \
  jq -r '.[].number' | \
  xargs -I {} gh issue edit {} --add-labels "priority:low"
```

## Using the GitHub REST API (when gh is not available)

```bash
# List issues via REST API
curl -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/julython/julython.org/issues?state=open&per_page=100"

# Create an issue
curl -X POST -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/julython/julython.org/issues" \
  -d '{"title":"New issue","body":"Issue body","labels":["bug"]}'

# Update an issue
curl -X PATCH -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/julython/julython.org/issues/42" \
  -d '{"state":"closed","labels":["bug","resolved"]}'

# Search issues (more powerful filters than gh)
curl -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/search/issues?q=repo:julython/julython.org+is:issue+label:bug+state:open&sort:comments&order:desc"
```

**Token setup (REST API):**
```bash
export GITHUB_TOKEN=ghp_your_personal_access_token
# Generate at: https://github.com/settings/tokens (scope: repo)
```

## Useful Patterns

### Issue number from title (when you know the title)
```bash
gh issue list --repo julython/julython.org --search "title:your title here" --json number,title
```

### Get issues by assignee across milestones
```bash
gh issue list --repo julython/julython.org --assignee "alice" --json number,title,milestone,state
```

### Generate an issue summary report
```bash
gh issue list --repo julython/julython.org --state all --json number,title,state,labels,assignees,comments,createdAt | \
  jq '[group_by(.state)[] | {state: .[0].state, count: length, first: .[0].number}]'
```

### Link related issues (duplicate/close references)
```bash
gh issue edit 42 --body "Duplicate of #100. See also #205."
gh issue close 43 --close-as-duplicate 100
```

## Work on a Milestone

When the user says "work on milestone X" or "continue milestone X":

1. **List open issues in the milestone**:
   ```bash
   gh issue list --repo julython/julython.org --milestone "<milestone-name>" \
     --json number,title,state,assignee,labels
   ```
   Present a numbered list to the user.

2. **Ask the user which issue to work on** — do NOT randomly pick one.
   Show the list, let them pick.

3. **View and work on the selected issue**:
   ```bash
   gh issue view 42
   ```

4. **Close it when done**:
   ```bash
   gh issue close 42
   ```

### Full workflow example

```bash
# Step 1: List open issues
gh issue list --repo julython/julython.org --milestone "player-boards" \
  --json number,title,state,assignee,labels | \
  jq -r '[.[] | select(.state=="open")] | sort_by(.number) | to_entries[] | "  \(.key + 1). #\(.value.number): \(.value.title)"'
# Output:
#   1. #180: Task 0.5: Add owner field to projects + enforce public-only
#   2. #181: Task 1: Add 3 board FK columns to Player
#   ...

# Step 2: User picks #180, agent views it
gh issue view 180
# (agent reads the issue, understands what to implement)

# Step 3: User finishes, agent closes
gh issue close 180
```

### Reference GitHub docs
- `gh issue --help` — full CLI reference
- [GitHub Issues docs](https://docs.github.com/en/issues/tracking-your-work-with-issues)
- [GitHub REST API: Issues](https://docs.github.com/en/rest/issues)
