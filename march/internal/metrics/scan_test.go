package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// always403RT implements http.RoundTripper and returns 403 without network I/O.
type always403RT struct{}

func (always403RT) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader(`{"message":"Forbidden"}`)),
		Header:     make(http.Header),
	}, nil
}

func TestClient_doGraphQL_403_returns_ErrGitHubForbidden(t *testing.T) {
	t.Parallel()

	c := &Client{httpClient: &http.Client{Transport: always403RT{}}, token: "tok"}
	_, err := c.doGraphQL(context.Background(), `query { __typename }`)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrGitHubForbidden) {
		t.Fatalf("expected ErrGitHubForbidden, got %v", err)
	}
}

func TestClient_fetchRecursiveTree_403_returns_ErrGitHubForbidden(t *testing.T) {
	t.Parallel()

	c := &Client{httpClient: &http.Client{Transport: always403RT{}}, token: "tok"}
	_, _, err := c.fetchRecursiveTree(context.Background(), "o", "r", "treesha")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrGitHubForbidden) {
		t.Fatalf("expected ErrGitHubForbidden, got %v", err)
	}
}

func TestWrappedPass1Error_is_ErrGitHubForbidden(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("pass 1 tree fetch: %w", ErrGitHubForbidden)
	if !errors.Is(err, ErrGitHubForbidden) {
		t.Fatalf("wrapped error should unwrap to ErrGitHubForbidden")
	}
}
