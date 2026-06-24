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
	// Prefixed match (production: OAuth/webhook users have gh-/gl- prefixes)
	prefix := "gh-"
	if project.Service == "gitlab" {
		prefix = "gl-"
	}
	projectOwner := prefix + project.Owner
	if strings.EqualFold(user.Username, projectOwner) {
		return true
	}
	// Direct match for backward compatibility (test/unprefixed users)
	return strings.EqualFold(user.Username, project.Owner)
}
