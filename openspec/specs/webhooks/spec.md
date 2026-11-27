# Webhooks

## Purpose

Receive GitHub push webhooks, record commits and projects, and feed them into game scoring after server-side analysis.

## Requirements

### Requirement: Ping and event filtering

The system SHALL respond to GitHub ping events and SHALL ignore webhook events that are not push events.

#### Scenario: Ping event

- **WHEN** a ping event is received
- **THEN** the system responds with a plain-text acknowledgement

#### Scenario: Non-push event

- **WHEN** a non-push event is received
- **THEN** the event is acknowledged and skipped

### Requirement: Payload decoding

The system SHALL accept JSON and form-encoded payloads, including smee.io-wrapped JSON payloads.

#### Scenario: Smee wrapped payload

- **WHEN** a JSON body contains a wrapper with a payload string
- **THEN** the inner payload is unwrapped and processed

### Requirement: Branch and force-push filtering

The system SHALL process only pushes to the repository default branch and SHALL skip force pushes.

#### Scenario: Feature branch push

- **WHEN** a push targets a non-default branch
- **THEN** the push is skipped

#### Scenario: Force push

- **WHEN** a push is marked as forced
- **THEN** the push is skipped

### Requirement: Project upsert

The system SHALL create or update the project record from the repository payload, keyed by repository ID.

#### Scenario: Inactive project

- **WHEN** the upserted project is inactive
- **THEN** the push is skipped

### Requirement: Commit validation and deduplication

The system SHALL skip low-quality commits (very short, WIP, merge, or CI-skipped messages) and commits whose hash already exists.

#### Scenario: Duplicate commit

- **WHEN** a commit hash already exists in the database
- **THEN** the commit is skipped

### Requirement: User resolution

The system SHALL resolve a commit author to a user by owner username, then by email identifier, creating a user for personal repositories when no match exists.

#### Scenario: Organization repository author

- **WHEN** a commit in an organization repository has no matching user
- **THEN** the commit is recorded without creating a new user

### Requirement: Analysis and scoring pipeline

The system SHALL run a server-side L1 scan for configured public repositories and SHALL score created commits after the scan completes.

#### Scenario: Scan not configured

- **WHEN** no analysis token is configured or the project is private
- **THEN** the scan is skipped and commits are scored directly