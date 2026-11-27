# Profile

## Purpose

Give authenticated users a profile overview, GitHub webhook management, and account settings.

## Requirements

### Requirement: Profile overview

The system SHALL show the signed-in user's username, name, avatar, and verified emails, redirecting unauthenticated visitors to login.

#### Scenario: Unauthenticated overview

- **WHEN** an unauthenticated request hits the profile overview
- **THEN** the user is redirected to login

### Requirement: Webhook repository listing

The system SHALL list the user's GitHub repositories with pagination and indicate which already have an active Julython webhook.

#### Scenario: Missing GitHub token

- **WHEN** the user has no stored GitHub token
- **THEN** the repository list request is rejected as unauthorized

### Requirement: Add webhook

The system SHALL create a GitHub webhook on a selected repository pointing at the configured webhook URL for push events.

#### Scenario: Missing owner or repo

- **WHEN** an add-webhook request omits owner or repo
- **THEN** a bad-request response is returned

### Requirement: Delete webhook

The system SHALL delete a previously created GitHub webhook for a repository.

#### Scenario: Repository not found

- **WHEN** the repository for a delete request cannot be found
- **THEN** a not-found response is returned

### Requirement: Settings update

The system SHALL update the user's display name from form or JSON input with validation.

#### Scenario: Empty name

- **WHEN** the submitted name is empty
- **THEN** the update is rejected with a validation message

#### Scenario: Name too long

- **WHEN** the submitted name exceeds 120 characters
- **THEN** the update is rejected with a validation message