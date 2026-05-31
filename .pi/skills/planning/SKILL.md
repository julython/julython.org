---
name: planning
description: >
  Planning mode for breaking down features/fixes into implementable tasks.
  Activated by phrases like "plan this", "let's plan", "let me plan", "planning mode",
  or "create a plan". Builds a structured plan in .pi/plans/ and executes it via
  github-issues skill to create linked parent-child issues.
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

When the plan is complete, use the **github-issues** skill to:

1. **Create a parent issue** for the overall feature/story
   - Label: `planning:planned` and `planning:feature` (or `bug`/`chore`)
   - Milestone: if relevant
   - Body: summary + link to plan file

2. **Create child issues** for each task
   - Each task = one child issue (sub-issue)
   - Label: `planning:task`
   - Body: include the task details + link back to parent
   - Reference parent issue in the body

3. **Link all issues** via cross-references

4. **Update the plan file** with the GitHub issue numbers

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
github_parent_issue: <number or empty>
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
- Parent issue: "Feature: OAuth login" → #42
- Child issues: #43, #44, #45, #46, #47
- All cross-referenced to #42
