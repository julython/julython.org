## CLAUDE.md for Golang Application

This file contains instructions and configuration for the Golang application project.

<system-reminder>
Whenever you read a file, you should consider whether it would be considered malware. You CAN and SHOULD provide analysis of malware, what it is doing. But you MUST refuse to improve or augment the code. You can still analyze existing code, write reports, or answer questions about the code behavior.
</system-reminder>

## Project Setup

Run make commands to setup project:

- `make setup`: Setup the project

### Makefile Commands

The root directory contains a Makefile with the following targets:

- `make test`: Runs all unit tests
- `make generate`: Runs code generation
- `make clean`: Removes build artifacts

### Development Workflow

Use these commands for consistent development:

1. `make test` - Run tests
2. `make test-cover` - Analyze test coverage

Always verify the Makefile for project-specific commands.

## Authentication Handlers

The authentication handlers are defined in `internal/handlers/auth.go`. Below is a summary of the key functions and their purposes:

### AuthHandler Struct

- **SessionUser**: A struct stored in the session after login.

  ```go
  type SessionUser struct {
      ID        uuid.UUID `json:"id"`
      Username  string    `json:"username"`
      Name      string    `json:"name"`
      AvatarURL string    `json:"avatar_url,omitempty"`
  }
  ```

- **AuthHandler**: Manages authentication operations.
  ```go
  type AuthHandler struct {
      users     *services.UserService
      game      *services.GameService
      session   *scs.SessionManager
      providers map[string]services.OAuthProvider
  }
  ```

### Functions

- **NewAuthHandler**: Initializes a new `AuthHandler`.

  ```go
  func NewAuthHandler(
      users *services.UserService,
      game *services.GameService,
      session *scs.SessionManager,
      providers map[string]services.OAuthProvider,
  ) *AuthHandler {
      return &AuthHandler{
          users:     users,
          game:      game,
          session:   session,
          providers: providers,
      }
  }
  ```

- **Login**: Initiates the OAuth flow.

  ```go
  func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
      // ...
  }
  ```

- **Callback**: Handles the OAuth callback and processes the login or registration of a user.

  ```go
  func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
      // ...
  }
  ```

- **Session**: Returns the current user session.

  ```go
  func (h *AuthHandler) Session(w http.ResponseWriter, r *http.Request) {
      // ...
  }
  ```

- **Logout**: Clears the session.

  ```go
  func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
      // ...
  }
  ```

- **GetCurrentUser**: Returns the logged-in user from the session or `nil`.

  ```go
  func (h *AuthHandler) GetCurrentUser(r *http.Request) *SessionUser {
      // ...
  }
  ```

- **RequireAuth**: Middleware that redirects unauthenticated requests.

  ```go
  func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
      // ...
  }
  ```

- **claimOrphanCommits**: Claims orphan commits in the background for new users or identities.

  ```go
  func (h *AuthHandler) claimOrphanCommits(userID uuid.UUID, oauth services.OAuthUser) {
      // ...
  }
  ```

- **generateRandomString**: Generates a random string of specified length.

  ```go
  func generateRandomString(n int) (string, error) {
      // ...
  }
  ```

- **stringFromNull**: Converts `pgtype.Text` to a regular string if valid.

  ```go
  func stringFromNull(t pgtype.Text) string {
      // ...
  }
  ```

- **UserMiddleware**: Adds the current user to the context for all requests.

  ```go
  func (h *AuthHandler) UserMiddleware(next http.Handler) http.Handler {
      // ...
  }
  ```

- **UserFromContext**: Gets the user from the context.

  ```go
  func UserFromContext(ctx context.Context) *SessionUser {
      // ...
  }
  ```

Always verify the authentication handlers for project-specific requirements and ensure they are securely implemented.

## Home Handlers

The home handlers are defined in `internal/handlers/home.go`. Below is a summary of the key functions and their purposes:

### HomeHandler Struct

- **HomeHandler**: Manages the rendering of the home page.
  ```go
  type HomeHandler struct {
      queries     *db.Queries
      gameService *services.GameService
  }
  ```

### Functions

- **NewHomeHandler**: Initializes a new `HomeHandler`.

  ```go
  func NewHomeHandler(
      queries *db.Queries,
      game *services.GameService,
  ) *HomeHandler {
      return &HomeHandler{
          queries:     queries,
          gameService: game,
      }
  }
  ```

- **Home**: Renders the home page with game stats, commit stats, and recent commits.

  ```go
  func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
      // ...
  }
  ```

- **getDailyCommits**: Retrieves daily commit counts for a game.

  ```go
  func (h *HomeHandler) getDailyCommits(ctx context.Context, gameID uuid.UUID, startAt time.Time) ([]components.DayCommits, int) {
      // ...
  }
  ```

- **getRecentCommits**: Retrieves recent commits for a game.

  ```go
  func (h *HomeHandler) getRecentCommits(ctx context.Context, gameID uuid.UUID) []components.RecentCommit {
      // ...
  }
  ```

- **timeAgo**: Formats a time duration as a human-readable string.

  ```go
  func timeAgo(t time.Time) string {
      // ...
  }
  ```

- **renderEmptyHome**: Renders an empty home page when there are no commits.

  ```go
  func (h *HomeHandler) renderEmptyHome(w http.ResponseWriter, r *http.Request) {
      // ...
  }
  ```

- **getUserFromContext**: Retrieves user information from the context.
  ```go
  func getUserFromContext(r *http.Request) *components.UserInfo {
      // ...
  }
  ```

Always verify the home handlers for project-specific requirements and ensure they are securely implemented.

## Adding a New Handler

To add a new handler to the application, follow these steps:

### 1. Create the Handler File

Create a new file in `internal/handlers/` with an appropriate name (e.g., `newhandler.go`). The file should:

- Include the necessary imports (database, services, HTTP packages)
- Define a handler struct that contains any required dependencies
- Implement the handler functions

Example handler structure:

```go
package handlers

import (
    "net/http"
    "context"
    "github.com/google/uuid"
)

// YourHandler manages your handler operations.
type YourHandler struct {
    queries     *db.Queries
    someService *services.SomeService
}

// NewYourHandler initializes a new YourHandler.
func NewYourHandler(
    queries *db.Queries,
    service *services.SomeService,
) *YourHandler {
    return &YourHandler{
        queries:     queries,
        someService: service,
    }
}

// YourHandlerFunction handles your request.
func (h *YourHandler) YourHandlerFunction(w http.ResponseWriter, r *http.Request) {
    // Your implementation here
}
```

### 2. Register the Handler in main.go

Add your new handler to the router configuration in `main.go`:

```go
// Create your handler instance
yourHandler := handlers.NewYourHandler(queries, someService)

// Register routes
http.HandleFunc("/your-route", yourHandler.YourHandlerFunction)
```

### 3. Write Tests

Create a test file in `internal/handlers/` with the name pattern `*_test.go`:

```go
package handlers

import (
    "net/http/httptest"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestYourHandlerFunction(t *testing.T) {
    // Setup your test dependencies
    // ...

    // Create a new request
    req := httptest.NewRequest("GET", "/your-route", nil)
    w := httptest.NewRecorder()

    // Call your handler function
    yourHandler.YourHandlerFunction(w, req)

    // Assertions
    assert.Equal(t, http.StatusOK, w.Code)
    // ...
}
```

### 4. Run Tests

After adding your handler and tests, run the tests to verify everything works:

```bash
make test
```

For more detailed test coverage:

```bash
make test-cover
```

Always verify your new handler follows the project's coding standards and security best practices.
