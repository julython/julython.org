package handlers

import (
	"testing"

	"july/internal/db"

	"github.com/stretchr/testify/assert"
)

func TestSlugOwner(t *testing.T) {
	tests := []struct {
		name      string
		slug      string
		wantOwner string
		wantErr   bool
	}{
		{"standard gh slug", "gh-rmyers-my-cool-repo", "rmyers", false},
		{"gl prefix", "gl-gitlab-org-goland", "gitlab", false},
		{"gh with hyphens in repo", "gh-jane-test-repo-name", "jane", false},
		{"gh with hyphens in owner", "gh-jane-doe-repo", "jane", false},
		{"too few parts", "gh-no-owner", "no", false},
		{"single part", "onlyone", "", true},
		{"two parts", "gh-two", "", true},
		{"empty slug", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := slugOwner(tc.slug)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantOwner, got)
		})
	}
}

func TestCanEditProject(t *testing.T) {
	admin := &db.User{Username: "admin", Role: "admin"}
	regular := &db.User{Username: "alice", Role: "user"}
	other := &db.User{Username: "bob", Role: "user"}

	tests := []struct {
		name    string
		user    *db.User
		project db.Project
		want    bool
	}{
		{"admin can edit any project", admin, db.Project{Slug: "gh-alice-some-repo"}, true},
		{"owner can edit", regular, db.Project{Slug: "gh-alice-my-project"}, true},
		{"non-owner cannot edit", other, db.Project{Slug: "gh-alice-my-project"}, false},
		{"case-insensitive owner match", &db.User{Username: "Alice", Role: "user"}, db.Project{Slug: "gh-alice-my-project"}, true},
		{"invalid slug denies access", regular, db.Project{Slug: "invalid-slug"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := canEditProject(tc.user, tc.project)
			assert.Equal(t, tc.want, got)
		})
	}
}
