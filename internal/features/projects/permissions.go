package projects

import (
	"strings"

	"july/internal/db"
)

// canEditProject returns true if the user is an admin or the project owner.
func canEditProject(user *db.User, project db.Project) bool {
	if user.Role == "admin" {
		return true
	}
	return strings.EqualFold(user.Username, project.Owner)
}
