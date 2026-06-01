package projects

import (
	"testing"

	"july/internal/db"

	"github.com/stretchr/testify/assert"
)

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
		{"admin can edit any project", admin, db.Project{Owner: "alice"}, true},
		{"owner can edit", regular, db.Project{Owner: "alice"}, true},
		{"non-owner cannot edit", other, db.Project{Owner: "alice"}, false},
		{"case-insensitive owner match", &db.User{Username: "Alice", Role: "user"}, db.Project{Owner: "alice"}, true},
		{"empty owner denies access", regular, db.Project{Owner: ""}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := canEditProject(tc.user, tc.project)
			assert.Equal(t, tc.want, got)
		})
	}
}
