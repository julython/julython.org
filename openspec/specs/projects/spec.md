# Projects

## Purpose

List and detail registered projects, show their analysis board and game activity, and let owners trigger re-analysis.

## Requirements

### Requirement: Project listing

The system SHALL list active projects with search, service filtering, and cursor-based pagination.

#### Scenario: Search filter

- **WHEN** the list is requested with a search term
- **THEN** only matching active projects are returned

#### Scenario: Cursor pagination

- **WHEN** more results exist than the page size
- **THEN** a next cursor is provided pointing at the last returned project

### Requirement: Project detail

The system SHALL show a project's metadata, analysis board, game activity summary, and recent commits.

#### Scenario: Unknown slug

- **WHEN** a project detail is requested for a slug that does not exist
- **THEN** a not-found response is returned

### Requirement: Analysis board

The system SHALL render an analysis board of metric tiles with levels and scores and SHALL compute earned points from score times level.

#### Scenario: Metric tiles

- **WHEN** analysis metrics exist for a project
- **THEN** a tile is shown per metric with its level and score

### Requirement: Edit permission

The system SHALL allow editing or rescanning a project only for an admin or the project owner.

#### Scenario: Non-owner edit attempt

- **WHEN** a non-owner without admin role attempts to rescan a project
- **THEN** the rescan control is not offered

### Requirement: L1 rescan

The system SHALL allow an owner to trigger a new L1 analysis scan for an eligible GitHub project.

#### Scenario: Private project rescan

- **WHEN** the project is private
- **THEN** the rescan is disabled with a private reason