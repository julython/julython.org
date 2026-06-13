---
name: webhook-placeholder-users
status: planning
created: 2026-06-12
modified: 2026-06-12
github_milestone: 3
github_milestone_title: webhook-placeholder-users
github_issue:
---

# Webhook Placeholder Users

## Goal
Auto-create user records from GitHub commit webhooks so that every commit has a user association, and when those users later sign up via OAuth, their account is found and upgraded with real OAuth data.

## Context
- Webhook handler processes push events, extracts commit author info (name, email, GitHub username), and creates commit records
- Currently, `processCommits()` matches commits by `email:<address>` identifier, but private/unverified emails have no identifier → commits get `user_id = NULL` (orphaned)
- OAuth login/registration creates users and links `github:<id>` identifiers (numeric)
- There's an existing `ClaimOrphanCommits()` that links commits by email
- GitHub webhook commits include `Author.Username` (the GitHub login) — this field is currently ignored
- GitHub usernames are globally unique and stable
- Git config emails are mostly unique in practice for a real project
- Projects are owned by the repo owner (created via webhook); committers can be different users who get credit for commits but don't own the project
- Players only get boards for commits they personally made, not for projects they contribute to
- The only login mechanism is OAuth — there's no password login

## Approach
- **No database migration** — keep existing `email` column on commits
- Create users from webhooks and add their email as an (unverified) identifier
- Lookup chain: `github:<id>` → `email:<address>` → `username = githubUsername`
- OAuth flow: if user found by `github:<id>`, upgrade it; if user found by `username`, add `github:<id>` to it

## User records created by webhooks:
- `username` = `Author.Username` (GitHub login, e.g. "rmyers")
- `name` = `Author.Name` (from git config, e.g. "Robert Myers")
- `avatar_url` = `Author.AvatarURL` (from webhook data)
- Unverified `email:<address>` identifier (linked from commit)
- No `github:<id>` identifier (added at OAuth signup)

## OAuth flow for existing webhook users:
1. Check `github:<numeric_id>` identifier (existing behavior)
2. **NEW:** If not found, check if any user has `username = oauthUsername` → if found, add `github:<numeric_id>` to that user
3. If neither, create new user (existing behavior)

## Tasks

### Task 1: Webhook handler creates users from commits using multi-stage lookup
- **Status:** `pending`
- **GitHub Issue:** #206
- **Description:** Modify `processCommits()` in the webhook handler to look up or create a user for each commit using a multi-stage lookup. For each commit author:

  1. Check if user exists with `github:<numeric_id>` identifier (current OAuth users)
  2. If not, check if user exists with `email:<address>` identifier (existing flow — private/unverified emails)
  3. If not, look up by `username = githubUsername` (NEW — catches users who committed but didn't sign up)
  4. If not found, create a new user with `username = githubUsername`, `name = Author.Name`, `avatar_url = Author.AvatarURL`
  5. If a user was found/created in step 3 or 4 (no `email:<address>` or `github:<id>` matched), add `email:<address>` as an unverified identifier
  6. Link the commit to this user
  7. On every commit, update the user's name and avatar (from webhook data)

- **Files to change:**
  - `internal/webhooks/github.go` — Add `AvatarURL string` to `GitHubAuthor` struct, remove `Email` from `GitHubAuthor` (or keep for reference only), add `getOrCreateUserForCommit()` method
  - `internal/db/queries/users.sql` — Add query if needed for username-based user lookup
- **Dependencies:** None (no migration)
- **Notes:** 
  - The lookup chain is: `github:<id>` → `email:<address>` → `username = githubUsername` → create new user
  - Step 5 is the key addition: if a user was found only by `username = githubUsername` (i.e., they committed but never signed up), we add their email as an unverified identifier so future commits can find them by email too
  - On every commit, call `UpdateUser` to refresh `name` and `avatar_url` (coalesced with existing values) so the user record stays fresh

### Task 2: Update OAuth login/registration to handle webhook-created users
- **Status:** `pending`
- **GitHub Issue:** #207
- **Description:** Modify `OAuthLoginOrRegister()` in `internal/services/users.go` to handle webhook-created users. The existing code checks `github:<id>` first (current OAuth users). After finding a user (whether from OAuth or webhooks), the OAuth flow should:
  - Update the user's name and avatar from OAuth data (coalesce with existing)
  - Add verified email identifiers from OAuth
  - If no user was found (new OAuth account), check if any user has `username = oauthUsername` (webhook user) and add `github:<id>` identifier to them
- **Files to change:** `internal/services/users.go`
  - In `OAuthLoginOrRegister()`, after the `github:<id>` check, add a fallback: look up by `username = oauthUsername`, if found, add `github:<numeric_id>` identifier to that user
  - The name/avatar update is already done when a user is found — just make sure it happens for webhook users too
- **Dependencies:** Task 1
- **Notes:** 
  - The webhook flow (Task 1) already handles users who have a `github:<id>` identifier — they're found in step 1 and get upgraded
  - The new path handles users who only exist as a `username` (created by webhooks) — when they sign up, we find them by username and add their `github:<id>`

### Task 3: Update tests for the new multi-stage lookup flow
- **Status:** `pending`
- **GitHub Issue:** #208
- **Description:** Add/update tests:
  1. Webhook creates a user from a commit (no existing account) — user has `username`, unverified `email:<address>` identifier
  2. Same user commits again → same user record (not duplicate)
  3. Same user commits from different email → same user (matched by `email:<address>` or `username`)
  4. OAuth login finds user by `github:<id>` and upgrades (existing behavior preserved)
  5. OAuth login finds webhook-created user by `username` and adds `github:<id>`
  6. Avatar/name are updated on each commit

- **Files to change:** `internal/webhooks/github_test.go`, `internal/auth/auth_integration_test.go`
- **Dependencies:** Task 1, Task 2
- **Notes:** Existing `processCommits()` tests that use `email` matching may need slight adjustment (they already work for `email:<address>` identifiers, but we're adding `username` matching as an additional path).
