package handlers

import (
	"fmt"
	"strings"

	"july/internal/db"
)

// canEditProject returns true if the user is an admin or the slug owner.
// Slug format: {service_prefix}-{username}-{repo...}
// e.g. "gh-rmyers-my-cool-repo" → owner is "rmyers"
func canEditProject(user *db.User, project db.Project) bool {
	if user.Role == "admin" {
		return true
	}
	owner, err := slugOwner(project.Slug)
	if err != nil {
		return false
	}
	return strings.EqualFold(user.Username, owner)
}

// slugOwner parses the owner username out of a project slug.
// Returns an error if the slug doesn't contain at least a prefix and owner.
func slugOwner(slug string) (string, error) {
	parts := strings.SplitN(slug, "-", 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("slug %q has no owner segment", slug)
	}
	return parts[1], nil
}
