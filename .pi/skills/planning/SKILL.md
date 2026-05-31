---
name: planning
description: >
  Planning mode for breaking down features/fixes into implementable tasks.
  Activated by phrases like "plan this", "let's plan", "let me plan", "planning mode",
  or "create a plan". Builds a structured plan in .pi/plans/ and executes it via
  github-issues skill to create a milestone (epic), parent feature issue, and
  child task issues all under that milestone.
---

# Planning Skill

Break down complex work into well-scoped, self-contained tasks ready for GitHub issues.

## Activation

This skill activates when the user says things like:
- "plan this"
- "let's plan"
- "planning mode"
- "create a plan for X"
- "let me plan this out"

## Workflow

### 1. Discovery & Clarification

Ask clarifying questions to understand:
- **What** are we building or fixing? (feature, bug, refactor, etc.)
- **Why** does this matter? (user value, technical necessity)
- **Constraints** (time, scope, dependencies, blocking items)
- **Unknowns** (things that need investigation before planning)
- **Success criteria** (how do we know it's done?)

Ask **one question at a time** — don't dump a questionnaire. Be conversational.

Use the existing codebase (`read`, `ls`) to inform your questions. Look at similar features, existing patterns, and architecture before asking about things you could discover yourself.

### 2. Build the Plan

Create (or update) a plan file at `.pi/plans/<plan-name>.md` using the template below.

Plan files are written with UTF-8 encoding using LF line endings and 2-space indentation.

### 3. Review the Plan

Walk the user through the plan. Ask about:
- Task boundaries (are they too big or too small?)
- Dependencies (are they correct?)
- Missing tasks (are there gaps?)
- Complexity estimates

Revise until the user says "looks good."

### 4. Execute to GitHub

When the plan is complete, decide: **is this a single story or multi-story feature?**

**Single-story plans** (1-3 simple tasks, no complex dependencies):
- Create one issue labeled `planning:task`
- Assign to an existing milestone if relevant, otherwise leave unassigned
- Body: include full task details (description, files to change, dependencies, complexity, notes)
- No milestone needed unless there are multiple related plans sharing one

**Multi-story features** (4+ tasks, or tasks with dependencies):
- **Ensure milestone exists** (the "epic" — no separate parent issue needed)
  - List existing milestones: `gh api repos/julython/julython.org/milestones --state open --jq '.[] | {number, title}'`
  - If it doesn't exist, create it: `gh api repos/julython/julython.org/milestones -X POST -d '{"title":"<plan-name>","state":"open"}'`
  - If it does exist, update its description: `gh api repos/julython/julython.org/milestones/{number} -X PATCH -f "title=<name>" -f "description=<goal+scope>"`
  - The milestone is the epic — its description contains the feature summary
  - The milestone name should be kebab-case, matching the plan name

2. **Create task issues** for each task
   - Each task = one issue
   - Label: `planning:task`
   - For multi-story: assign to the created milestone (by name)
   - Body: include the task details (description, files to change, dependencies, complexity, notes)
   - Reference other tasks by number in the body if needed

3. **Link all issues** via cross-references (parent task references children, children reference parent)

4. **Update the plan file** with the milestone info and issue numbers

### 5. Track Progress

Update the plan file as tasks are worked on:
- Mark tasks done
- Move task labels from `planning:task` → `planning:in-progress` → `planning:done`
- Link PRs when they're opened

## Plan File Template

```markdown
---
name: <slug-name>
status: active  # active | planning | ready | done
created: <YYYY-MM-DD>
modified: <YYYY-MM-DD>
github_milestone: <number or empty>
github_milestone_title: <milestone name>
github_issue: <number or empty>  # for single-story plans
---

# <Feature Name>

## Goal
One-paragraph description of what this achieves and why.

## Context
- Current state / problem
- Technical constraints (framework, architecture, existing patterns)
- Dependencies (other teams, services, features)

## Requirements
<!-- Filled during discovery phase -->

## Non-goals
<!-- Explicitly what we're NOT doing (prevents scope creep) -->

## Tasks

### Task 1: <Task Title>
- **Status:** `pending` (pending | in-progress | done)
- **GitHub Issue:** N/A
- **Description:** What this specific task accomplishes
- **Files to change:** `path/to/file.go`, `path/to/file.new.go`
- **Dependencies:** None (or link to Task 2)
- **Complexity:** Low (1hr) | Medium (4hr) | High (1d+)
- **Notes:** Any gotchas, patterns to follow, migration steps

### Task 2: <Task Title>
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** ...
- **Files to change:** ...
- **Dependencies:** Task 1
- **Complexity:** Medium
- **Notes:** ...
```


## Task Sizing Rules

Each task (child issue) should be **self-contained in a single PR**, meaning:

1. **Single responsibility** — one clear change, not "fix auth AND update tests"
2. **Runnable + testable** at every step — the app doesn't break between tasks
3. **Under ~4 hours** of implementation for junior devs (adjust for seniors)
4. **Includes its own tests** — no "add tests in final task"
5. **Clear files list** — know what files before you start

If a task is too big, break it further. A task is NOT "write documentation." It's "write documentation" only when docs are the actual deliverable and don't depend on code changes.

## Dependency Graph

- Tasks with no dependencies can be worked on in parallel
- Note blocking relationships in the `Dependencies` field
- A task is blocked if any dependency's `Status` is not `done`

## Plan File Convention

- Location: `.pi/plans/<slug-name>.md`
- Slug: kebab-case, derived from the feature name
- One plan file per feature/story
- Plan files are tracked in git

## Example

User: "Let's plan adding OAuth to the login page"

Agent goes through discovery, builds a plan like:

```
## Tasks

### Task 1: Add OAuth config to env and config module
### Task 2: Create OAuth callback handler (Go backend)
### Task 3: Add /auth/callback route and redirect logic
### Task 4: Update login page UI with "Login with Google" button
### Task 5: Write integration tests for OAuth flow
```

Then executes to GitHub:
- Milestone: "oauth-login" (created/updated with feature description)
- Task issues: #43, #44, #45, #46, #47 (all assigned to milestone, labeled `planning:task`)
- All cross-referenced, all grouped under the milestone
- The milestone description serves as the epic summary (no separate parent issue needed)

### Single-story plan

User: "Let's plan adding a canonical-tag command to git-get"

Agent discovers it's a simple one-off task, builds a plan:

```markdown
---
name: canonical-tag
status: active
created: 2026-05-31
github_issue: 150
---

## Task 1: Add canonical-tag command
- **Status:** `pending`
- **Description:** Make git-get output canonical repo paths (strip .git suffix, resolve redirects)
- **Files:** `cmd/git-get/canonical.go`, `internal/git/resolve.go`
- **Dependencies:** None
- **Complexity:** Low
```

Then executes to GitHub:
- One issue: #150 labeled `planning:task`
- Body: includes full description, files to change, complexity, notes
- No milestone needed (single story, no related work)
