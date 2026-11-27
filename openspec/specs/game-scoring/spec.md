# Game Scoring

## Purpose

Track commits and projects during an active Julython game and turn them into points, boards, and player scores.

## Requirements

### Requirement: Active game resolution

The system SHALL resolve the active game for a given point in time, falling back to the most recently ended game when none is active, and SHALL report no active game when neither exists.

#### Scenario: Active game at commit time

- **WHEN** a commit is processed and a game is active at the commit timestamp
- **THEN** the commit is attributed to that game

#### Scenario: No active game

- **WHEN** no game is active at the commit timestamp
- **THEN** the commit is ignored for scoring without error

### Requirement: Commit and project points

The system SHALL award the game's commit points for each valid commit and an additional project-points bonus the first time a project scores in that game.

#### Scenario: First commit to a project

- **WHEN** the first commit for a project is scored in a game
- **THEN** the project board receives commit points plus the project-points bonus

#### Scenario: Subsequent commit to a project

- **WHEN** a later commit for the same project is scored
- **THEN** the project board receives only commit points

### Requirement: Public projects only

The system SHALL exclude private projects from game scoring while still recording their commits.

#### Scenario: Private project commit

- **WHEN** a commit belongs to a private project
- **THEN** the commit is stored but no board or player points are awarded

### Requirement: Language boards

The system SHALL accumulate per-language points for each game from the languages detected in a commit.

#### Scenario: Commit with detected languages

- **WHEN** a commit has one or more detected languages
- **THEN** each language board for the game is credited with commit points

### Requirement: Player boards and verified points

The system SHALL assign each player up to three project boards per game and SHALL compute verified points from the player's commit count and assigned board totals.

#### Scenario: Player exceeds three boards

- **WHEN** a player already has three assigned boards and a new board becomes available
- **THEN** the new board is not assigned

#### Scenario: Verified points recalculation

- **WHEN** a player's boards or commit counts change
- **THEN** verified points are recomputed as commit count plus the sum of assigned board points

### Requirement: Orphan commit claiming

The system SHALL reassign commits with no user to a user when the commit author email matches one of the user's verified emails.

#### Scenario: Claim on login

- **WHEN** a user logs in with verified emails that match orphan commits in the active game
- **THEN** those commits are reassigned to the user and the player score is updated

### Requirement: Analysis adjustment

The system SHALL apply an AI analysis adjustment to a player's verified points and mark analysis complete.

#### Scenario: Apply adjustment

- **WHEN** an analysis adjustment is applied to a player
- **THEN** verified points equal potential points plus the adjustment and status becomes completed