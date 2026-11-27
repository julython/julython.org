# Leaderboard

## Purpose

Rank players, projects, and languages for the active or most recent game with paginated, HTMX-friendly views.

## Requirements

### Requirement: Game selection

The system SHALL display the active game, or the most recently ended game when none is active, and SHALL show an empty board when no game exists.

#### Scenario: No game

- **WHEN** no active or latest game exists
- **THEN** an empty leaderboard is rendered

### Requirement: Player leaderboard

The system SHALL rank players by verified points with rank, username, name, avatar, commit count, project count, and points.

#### Scenario: Paginated players

- **WHEN** the leaderboard is requested with an offset
- **THEN** ranks reflect the offset and pagination continues while more rows exist

### Requirement: Project leaderboard

The system SHALL rank projects for the game with points, potential points, verified points, commit count, and contributor count.

#### Scenario: Project ranking

- **WHEN** the project leaderboard is requested
- **THEN** projects are ordered by points for the selected game

### Requirement: Language leaderboard

The system SHALL rank languages for the game by points and commit count.

#### Scenario: Language ranking

- **WHEN** the language leaderboard is requested
- **THEN** languages are ordered by points for the selected game

### Requirement: HTMX partial rendering

The system SHALL render only the table fragment for HTMX requests and a full page otherwise.

#### Scenario: HTMX request

- **WHEN** the request carries the HTMX header
- **THEN** only the leaderboard table fragment is returned