# Authentication

## Purpose

Authenticate users via OAuth providers (GitHub) with a server-side session, and gate protected routes.

## Requirements

### Requirement: OAuth login initiation

The system SHALL begin an OAuth flow for a known provider by generating state and a PKCE verifier, storing them in the session, and redirecting to the provider authorization URL.

#### Scenario: Unknown provider

- **WHEN** a login request names a provider that is not configured
- **THEN** the request fails with a bad-request response

### Requirement: OAuth callback validation

The system SHALL validate the returned OAuth state against the session and SHALL reject callbacks with missing or mismatched state.

#### Scenario: State mismatch

- **WHEN** the callback state does not match the stored state
- **THEN** the callback is rejected and no session is created

### Requirement: Login or registration

The system SHALL exchange the authorization code, fetch user info, and either log in an existing user or register a new one, then store the user in the session.

#### Scenario: New user registration

- **WHEN** the OAuth user is not yet registered
- **THEN** a new user is created and the user is logged in

### Requirement: Orphan commit claim on registration

The system SHALL claim orphan commits for a newly registered user using the verified emails from the provider.

#### Scenario: Verified emails present

- **WHEN** a new user registers with verified emails
- **THEN** matching orphan commits are claimed in the background

### Requirement: Session and logout

The system SHALL expose the current session as JSON and SHALL destroy the session on logout, returning JSON for API clients and redirecting browsers.

#### Scenario: Unauthenticated session request

- **WHEN** the current user is requested without a session
- **THEN** a not-authenticated response is returned

### Requirement: Protected routes

The system SHALL provide middleware that redirects unauthenticated browser requests to login (preserving the target path) and returns unauthorized for JSON API requests.

#### Scenario: Browser hits protected route

- **WHEN** an unauthenticated browser request hits a protected route
- **THEN** the user is redirected to the provider login and returned to the original path afterward