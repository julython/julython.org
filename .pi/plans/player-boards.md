---
name: player-boards
status: active
created: 2026-05-31
modified: 2026-05-31
github_milestone: 2
github_milestone_title: player-boards
---

# Feature: Player Boards

## Goal
Give logged-in users a personalized view of 3 projects they're "playing" during Julython. Users who have at least one linked project (via webhook) get an automatic board selection. New users see onboarding docs. Boards display the project analysis grid (no chat/LLM). The bottom of the page shows the global commit feed. Uses a **two-tier scoring model on the existing `Board` object**: persistent `verified_points` (LLM) survive across games, per-game `points` (commits/project bonuses) reset per game via new Board rows.

## Context
- Julython is a monthly coding competition where commits earn points
- Currently: per-project **boards** and per-user **players** exist in `boards` and `players` tables
- Scoring: `commit_count × commit_points + project_count × project_points`, with `verified_points` overriding when > 0
- Webhooks: users add webhooks to repos on `/profile/webhooks` → commits flow into `projects` and `boards`
- **No explicit user-to-project linkage** — projects are discovered via webhooks but users aren't assigned boards
- **No `owner` field on `projects` table** — the owner is manually parsed from the repo URL in `processCommits()` (fragile: `strings.ReplaceAll(fullName, "/", "-")`)
- **No public enforcement** — private repos can currently enter the game via webhook (LLM analysis skips private repos but they still get points)
- The project page already has an analysis grid (metrics tiles, scoring bar, L1 rescan)
- **Two-tier scoring (on existing `Board` object):**
  - `points` (and `commit_count`, `project_count`) — **per-game scoring**, updated by `AddCommit` when a commit arrives during the active game. A new game creates a new Board row (new `game_id`), so this resets naturally.
  - `verified_points` — **persistent/LLM score**, set by AI analysis. This persists across games.
  - Total score per board = `boards.points + boards.verified_points`.

## Requirements
- [ ] Requirement: **Add `owner` field to `projects` table** — stored when webhook processes commit (already available in `GitHubCommit.Author.Username`) — replaces fragile URL parsing
- [ ] Requirement: **Migrate existing projects** — extract owner from existing repo URLs via migration seed script
- [ ] Requirement: **Enforce public-only for scoring** — only public repos earn game points (private repos still allowed for LLM analysis, just no game points)
- [x] Requirement: logged-in users only see player boards (non-logged-in users redirected to login or see a generic page)
- [ ] Requirement: new users (0 linked projects) see onboarding docs explaining how to add webhooks
- [x] Requirement: user boards automatically add the first 3 projects that get a commit during the game
- [x] Requirement: max 3 boards; user can swap out if they have > 3 linked projects
- [x] Requirement: boards use the same project-page grid (metrics tiles, scoring bars, commit summary) — no chat/LLM
- [x] Requirement: commits from all 3 boards shown below the board grid (showing: per-commit points + per-project bonus + verified)
- [ ] Requirement: two-tier scoring on existing `Board` object — persistent `verified_points` + per-game `points`, sum for total
- [x] Requirement: consistent sorting by total points (persistent + per-game)

## Non-goals
- No chatbot or LLM integration on the player board page (explicitly excluded by the user)
- No team-boards (Teams system exists but out of scope)
- No mobile responsiveness overhaul (use existing responsive patterns)
- No change to the webhook onboarding UX (already works — we just surface it for new users)
- Full score-reset implementation (just tests, per user: "6 months to figure this out")
- **No new `game_scores` table** — Board already has both per-game (`points`) and persistent (`verified_points`)

## Tasks

### Task 0.5: Add `owner` field to projects + enforce public-only
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** Two parts:
  - **a) Schema:** Add `owner VARCHAR NOT NULL DEFAULT ''` to the `projects` table. Populate it from `GitHubCommit.Author.Username` (already in webhook handler's `processCommits()` — replaces the fragile manual URL parsing). Write a migration for existing projects.
  - **b) Public enforcement:** When processing commits via webhook, only add commits to scoring if the project is public (`projects.is_public = true`). Private repos are allowed for LLM analysis (already handled in `scheduleL1Scan`) but excluded from game scoring.
  - Update `processCommits()` in `internal/webhooks/github.go` to set `owner` from the webhook payload (already there: `c.Author.Username`).
  - The `Project` model already has `IsPrivate bool` from migration `20260406`. This task populates `is_private` for existing projects too.
- **Files to change:** `migrations/<timestamp>_add_projects_owner.up.sql`, `migrations/<timestamp>_add_projects_owner.down.sql`, `migrations/<timestamp>_seed_projects_owner.up.sql`, `internal/webhooks/github.go`, `internal/db/models.go` (regenerate), optionally `internal/features/projects/projects.go` (display owner)
- **Dependencies:** None
- **Complexity:** Low-Medium
- **Notes:** The `GitHubCommit` struct in `webhooks/github.go` has `Author.Username` available — this is the correct source of truth. Existing projects with empty owner get migrated: parse the URL, extract owner. Check `is_private` from the webhook payload to populate for projects that came in as private.

### Task 1: Add 3 board FK columns to Player + rename concept
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** The `Player` table (already has `game_id` + `user_id`) gets 3 additional FK columns pointing to the 3 active project boards:
  - `board_1_id` (uuid, nullable, FK → boards)
  - `board_2_id` (uuid, nullable, FK → boards)
  - `board_3_id` (uuid, nullable, FK → boards)
  This replaces the `user_boards` concept — the player's active boards are 3 columns on the existing `Player` table. No separate table needed.
  Add SQLC queries: read 3 boards for a player, swap board position (update one of the 3 FKs), list all players with their 3 active boards.
- **Files to change:** `migrations/<timestamp>_add_player_boards.up.sql`, `migrations/<timestamp>_add_player_boards.down.sql`, `internal/db/queries/`, regenerate with `make sqlc`
- **Dependencies:** Task 0.5
- **Complexity:** Low
- **Notes:** The `Player` model (in `models.go`) gets 3 new fields: `Board1ID`, `Board2ID`, `Board3ID` (all `pgtype.UUID{Valid: false}` initially). On board assignment, `AssignBoards` updates the appropriate column. `SwapBoard` updates the FK directly. Simple, no JOIN needed.

### Task 2: Clarify two-tier scoring on existing `Board` object
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** No new table. The `Board` object already captures both tiers:
  - `points` (and `commit_count`, `project_count`) — **per-game scoring**, automatically updated by `AddCommit` when a commit arrives during the active game. This resets implicitly when a new game starts (a new Board row is created for the new game).
  - `verified_points` — **persistent/LLM score**, set by AI analysis. This is what persists across games.
  - Total score = `boards.points` (per-game commit/project) + `boards.verified_points` (persistent).
  The key insight: a new game creates a new `Board` row (new `game_id`), so per-game scoring is naturally isolated. No migration needed. Just document and use consistently in the player board display.
- **Files to change:** `internal/services/game.go` (display both tiers), `internal/features/game/boards.go` (handler aggregates both tiers when rendering boards)
- **Dependencies:** Task 1
- **Complexity:** Low
- **Notes:** The existing `AddCommit` in `game.go` already writes `points` and `commit_count` to Board. `verified_points` is set by the L1 scanner. We just read both and sum them when displaying player board scores. The leaderboard query already has `CASE WHEN verified_points > 0 THEN verified_points ELSE potential_points` — we may need to adjust it to SUM both when computing player totals.

### Task 3: Backend — PlayerBoard service
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** Create a `PlayerBoardService` that manages board assignment and score aggregation. Key methods:
  - `GetPlayerBoards(ctx, gameID, userID) → []BoardWithScores` — fetches the player's 3 active boards with combined (persistent + per-game) scores
  - `AssignBoards(ctx, gameID, userID) error` — auto-assigns up to 3 public projects that have commits (finds the Player record, sets the unfilled FK columns to matching Board records)
  - `SwapBoard(ctx, gameID, userID, boardNumber, projectID) error` — replaces one of the 3 board FKs
  - `GetTotalScore(ctx, gameID, userID) int` — persistent + per-game across all 3 active boards
  - `HasLinkedProjects(ctx, userID) bool` — does user have at least 1 project (for onboarding check)
- **Files to change:** `internal/services/playerboard.go`, `internal/services/playerboard_test.go`
- **Dependencies:** Tasks 1, 2
- **Complexity:** Medium
- **Notes:** `GetPlayerBoards` reads `board_1/2/3_id` from `Player` table, resolves each board's data (total = `boards.points + boards.verified_points`). `AssignBoards` finds the Player record for the game, checks which of the 3 columns are null, fills them with matching Board records. Filters to public projects only.

### Task 4: Backend — Player board handler (HTTP routes)
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** HTTP routes for the player board page:
  - `GET /boards` — player board page (logged-in users)
  - `GET /api/v1/boards` — JSON API for HTMX swaps
  - `POST /api/v1/boards/{position}/swap` — HTMX to swap a board (positions 1-3)
  - Redirect unauthenticated users to login
- **Files to change:** `internal/features/game/boards.go`, `internal/features/game/boards_test.go`, mount in `cmd/server/main.go`
- **Dependencies:** Task 3
- **Complexity:** Medium
- **Notes:** Handler needs GameService (active game) and PlayerBoardService. For 0-linked-projects users, render onboarding view.

### Task 5: Frontend — Player board page (templ)
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** Player board page template. Layout:
  - **Top:** 3 project boards side-by-side (grid, metrics tiles, scoring bars, commit summary, verified points)
  - **Bottom:** global commit feed from all 3 boards (time-sorted, showing: per-commit + per-project + verified points)
  - **0 linked projects:** onboarding docs (how to add webhooks, link to `/profile/webhooks`)
  - **> 3 boards:** "swap" buttons on each board to pick which 3 to show
  - Each board card: clickable link to `/projects/{slug}` (no chat/LLM)
  - Reuses project-page analysis grid styling (metric tiles, scoring bar, dash indicators)
- **Files to change:** `internal/features/game/boards.templ`, `internal/features/game/boards.go` (data types), `internal/i18n/locales/en.json` (strings: "PlayerBoards", "NoLinkedProjects", "HowToGetStarted", etc.)
- **Dependencies:** Task 4
- **Complexity:** High (biggest UI task)
- **Notes:** Reuse `projects.templ` components (analysis board grid, metric tiles). ~1/3 width per board on desktop, stack on mobile. Commit feed below reuses simplified activity feed pattern from `activity.templ`.

### Task 6: Frontend — Onboarding for new users
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** When a user has 0 linked projects, show an onboarding fragment:
  - "Welcome to Julython! To play, add a webhook to one of your projects:"
  - Link to `/profile/webhooks` (hint: "Go to your profile → Webhooks")
  - Optional: small "repo → webhook → Julython" diagram
  - Users with 1+ projects but 0 boards get: "Your boards will appear once your projects receive commits"
- **Files to change:** `internal/features/game/boards.templ` (onboarding template), `internal/i18n/locales/en.json`
- **Dependencies:** Task 5
- **Complexity:** Low
- **Notes:** Simple template. Reuse empty state pattern from other pages.

### Task 7: Backend — Score reset tests (no full implementation)
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** Add tests to `game_test.go` verifying:
  - A new game's Board rows start with `points = 0` (no carryover from old games)
  - Existing `boards.verified_points` from prior games on the same project are NOT affected (persistent — same project, different `game_id` rows keep their verified_scores)
  - Total score displays correctly: `total = boards.points + boards.verified_points`
  - (Optional) `ResetPlayerBoards(ctx, gameID)` helper called in `CreateGame` when `deactivateOthers=true` (clears the 3 board FKs on Player)
- **Files to change:** `internal/services/game_test.go`
- **Dependencies:** Task 1
- **Complexity:** Low
- **Notes:** Per user: "6 months to figure this out so do not over-rotate." Just tests. New games create fresh Board rows (no records until commits arrive). The 3 board FKs on `Player` are nullable so unassigned players are normal.

### Task 8: Frontend — Nav link to Boards page
- **Status:** `pending`
- **GitHub Issue:** N/A
- **Description:** Add "Boards" link to the navigation bar for logged-in users. Add next to existing tabs (Home, Activity, Leaders). Only visible to authenticated users.
- **Files to change:** `internal/components/layout/layout.templ` (add Boards tab)
- **Dependencies:** Task 5
- **Complexity:** Low
- **Notes:** Check how other nav items are rendered. Likely a simple addition to the tabs nav component.
