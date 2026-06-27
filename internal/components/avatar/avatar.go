package avatar

// AvatarSize controls the dimensions of AvatarFallback.
type AvatarSize string

const (
	AvatarSm AvatarSize = "sm" // 24x24
	AvatarMd AvatarSize = "md" // 32x32
	AvatarLg AvatarSize = "lg" // 40x40
)

// AvatarSizeClass returns the CSS class string for a given AvatarSize.
func AvatarSizeClass(size AvatarSize) string {
	switch size {
	case AvatarSm:
		return "w-8 h-8 rounded-full flex-shrink-0"
	case AvatarMd:
		return "w-10 h-10 rounded-full flex-shrink-0"
	case AvatarLg:
		return "w-10 h-10 rounded-full ring-2 ring-transparent group-hover:ring-july-500/50 transition-all"
	default:
		return "w-8 h-8 rounded-full flex-shrink-0"
	}
}

// AvatarCircleClass returns the CSS class string for the avatar circle.
func AvatarCircleClass(size AvatarSize) string {
	switch size {
	case AvatarSm:
		return "w-8 h-8 rounded-full bg-july-500/20 flex items-center justify-center text-july-400 text-sm flex-shrink-0"
	case AvatarMd:
		return "w-10 h-10 rounded-full bg-july-500/20 flex items-center justify-center text-july-400 text-sm flex-shrink-0"
	case AvatarLg:
		return "w-10 h-10 rounded-full bg-gradient-to-br from-july-500 to-purple-600 flex items-center justify-center text-white font-bold"
	default:
		return "w-8 h-8 rounded-full bg-july-500/20 flex items-center justify-center text-july-400 text-sm flex-shrink-0"
	}
}

// AvatarInitials returns initials from the display name (first letter of each
// word, up to 2), or a fallback character from the username if no display
// name is provided.
func AvatarInitials(username string, displayName ...string) string {
	if len(displayName) > 0 && displayName[0] != "" {
		return displayNameInitials(displayName[0])
	}
	return fallbackChar(username)
}

// displayNameInitials extracts up to 2 initials from a display name.
// Skips emoji and non-ASCII characters.
func displayNameInitials(displayName string) string {
	words := splitDisplayWords(displayName)
	if len(words) == 0 {
		return ""
	}
	initials := make([]byte, 0, 2)
	for _, w := range words {
		if len(initials) >= 2 {
			break
		}
		for _, r := range w {
			// Skip non-ASCII characters (emojis, etc.)
			if r > 127 {
				continue
			}
			initials = append(initials, byte(toUpper(r)))
			break
		}
	}
	if len(initials) == 0 {
		return ""
	}
	return string(initials)
}

// splitDisplayWords splits a display name into words by whitespace.
func splitDisplayWords(name string) []string {
	words := []string{}
	inWord := false
	var current []rune
	for _, r := range name {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if inWord {
				words = append(words, string(current))
				current = nil
				inWord = false
			}
		} else {
			current = append(current, r)
			inWord = true
		}
	}
	if inWord {
		words = append(words, string(current))
	}
	return words
}

// fallbackChar produces a useful character from the username:
// for gh- prefixed usernames, strip the prefix and split on hyphens
// to take the second segment's first character (uppercased);
// otherwise falls back to username[3] (the 4th character).
func fallbackChar(username string) string {
	// Only apply gh- prefix handling for GitHub usernames.
	if len(username) > 3 && username[:3] == "gh-" {
		afterPrefix := username[3:]
		parts := splitOnHyphen(afterPrefix)
		for _, p := range parts[1:] {
			for _, r := range p {
				return string(toUpper(r))
			}
		}
		// If the part after gh- is just digits (e.g., gh-12345),
		// no meaningful hyphen split found — fall through to default.
	}
	// Default: use the 4th character (index 3).
	if len(username) > 3 {
		return string(username[3])
	}
	// Truly short/empty username — just take first character.
	if len(username) > 0 {
		return string(toUpper(rune(username[0])))
	}
	return "?"
}

// splitOnHyphen splits a string on hyphens.
func splitOnHyphen(s string) []string {
	parts := []string{}
	var current []rune
	for _, r := range s {
		if r == '-' {
			parts = append(parts, string(current))
			current = nil
		} else {
			current = append(current, r)
		}
	}
	parts = append(parts, string(current))
	return parts
}

// toUpper converts a rune to its uppercase equivalent.
func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 'a' + 'A'
	}
	return r
}
