# Blog

## Purpose

Serve static Markdown blog posts rendered to HTML with syntax highlighting and Mermaid diagram support.

## Requirements

### Requirement: Post loading

The system SHALL load posts from embedded Markdown content and parse frontmatter for title, date, slug, and blurb.

#### Scenario: Invalid date

- **WHEN** a post has a date that does not parse
- **THEN** loading fails with an error identifying the file

### Requirement: Post ordering

The system SHALL return all posts sorted newest first by date.

#### Scenario: List posts

- **WHEN** all posts are requested
- **THEN** posts are ordered by descending date

### Requirement: Single post lookup

The system SHALL return a single post by slug.

#### Scenario: Unknown slug

- **WHEN** a post is requested by a slug that does not exist
- **THEN** an error is returned

### Requirement: Rendering features

The system SHALL render GFM Markdown with syntax highlighting and SHALL pass Mermaid fenced blocks through for client-side rendering.

#### Scenario: Mermaid block

- **WHEN** a post contains a Mermaid fenced block
- **THEN** the block is emitted as a Mermaid container and the post is flagged as having Mermaid content