package projects

import (
	"testing"

	"july/internal/db"

	"github.com/stretchr/testify/assert"
)

func TestCanEditProject(t *testing.T) {
	admin := &db.User{Username: "admin", Role: "admin"}
	regular := &db.User{Username: "gh-alice", Role: "user"}
	other := &db.User{Username: "gh-bob", Role: "user"}

	tests := []struct {
		name    string
		user    *db.User
		project db.Project
		want    bool
	}{
		{"admin can edit any project", admin, db.Project{Owner: "alice"}, true},
		{"owner can edit", regular, db.Project{Owner: "alice"}, true},
		{"non-owner cannot edit", other, db.Project{Owner: "alice"}, false},
		{"case-insensitive owner match", &db.User{Username: "GH-alice", Role: "user"}, db.Project{Owner: "alice"}, true},
		{"empty owner denies access", &db.User{Username: "gh-someone", Role: "user"}, db.Project{Owner: ""}, false},
		{"gitlab owner can edit", &db.User{Username: "gl-alice", Role: "user"}, db.Project{Owner: "alice", Service: "gitlab"}, true},
		{"non-gitlab-owner cannot edit", &db.User{Username: "gl-bob", Role: "user"}, db.Project{Owner: "alice", Service: "gitlab"}, false},
		{"github user cannot edit gitlab project", &db.User{Username: "gh-alice", Role: "user"}, db.Project{Owner: "alice", Service: "gitlab"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := canEditProject(tc.user, tc.project)
			assert.Equal(t, tc.want, got)
		})
	}
}
