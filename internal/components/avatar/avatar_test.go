package avatar

import "testing"

func TestDisplayNameInitials(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"two words", "Mark Smith", "MS"},
		{"single word", "Just", "J"},
		{"three words", "Alice Bob Charlie", "AB"},
		{"all caps", "JOHN DOE", "JD"},
		{"mixed case", "jOhN dOe", "JD"},
		{"empty string", "", ""},
		{"just whitespace", "   ", ""},
		{"single letter word", "a b c", "AB"},
		{"emoji", "👋 Hello", "H"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := displayNameInitials(tt.input)
			if got != tt.expected {
				t.Errorf("displayNameInitials(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDisplayNameInitials_ValidatedWithSplitDisplayWords(t *testing.T) {
	// Verify splitDisplayWords works as expected for edge cases.
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple", "Mark Smith", []string{"Mark", "Smith"}},
		{"triple", "A B C", []string{"A", "B", "C"}},
		{"empty", "", []string{}},
		{"just spaces", "   ", []string{}},
		{"trailing space", "Mark Smith ", []string{"Mark", "Smith"}},
		{"leading space", " Mark Smith", []string{"Mark", "Smith"}},
		{"double space", "Mark  Smith", []string{"Mark", "Smith"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitDisplayWords(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("splitDisplayWords(%q) length = %d, want %d", tt.input, len(got), len(tt.expected))
			} else {
				for i := range got {
					if got[i] != tt.expected[i] {
						t.Errorf("splitDisplayWords(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
					}
				}
			}
		})
	}
}

func TestFallbackChar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"github short", "gh-12345", "1"},
		{"github with name", "gh-john-doe", "D"},
		{"github single segment", "gh-12345", "1"},
		{"gh prefix, no hyphen after", "gh-user", "u"},
		{"normal username", "alice", "c"},
		{"normal short", "bob", "B"},
		{"normal three", "bob", "B"},
		{"normal two", "ab", "A"},
		{"normal one", "a", "A"},
		{"empty string", "", "?"},
		{"single char", "x", "X"},
		{"starts with gh but short", "gh", "G"},
		{"no gh prefix", "johnny", "n"},
		{"hyphenated no gh", "john-doe", "n"},
		{"just hyphens", "---", "-"},
		{"all hyphens but first empty", "---abc", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fallbackChar(tt.input)
			if got != tt.expected {
				t.Errorf("fallbackChar(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAvatarInitials(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		displayName []string
		expected    string
	}{
		{"no display name, gh user", "gh-12345", nil, "1"},
		{"no display name, normal", "alice", nil, "c"},
		{"with display name, two words", "gh-12345", []string{"Mark Smith"}, "MS"},
		{"with display name, single word", "gh-12345", []string{"Just"}, "J"},
		{"with display name, empty", "gh-12345", []string{""}, "1"},
		{"with display name, nil slice", "gh-12345", nil, "1"},
		{"with display name, multiple", "gh-12345", []string{"Alice Bob Charlie"}, "AB"},
		{"normal user with display", "alice", []string{"Alice Johnson"}, "AJ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AvatarInitials(tt.username, tt.displayName...)
			if got != tt.expected {
				t.Errorf("AvatarInitials(%q, %v) = %q, want %q", tt.username, tt.displayName, got, tt.expected)
			}
		})
	}
}

func TestSplitOnHyphen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"no hyphens", "hello", []string{"hello"}},
		{"single hyphen", "hello-world", []string{"hello", "world"}},
		{"multiple hyphens", "a-b-c-d", []string{"a", "b", "c", "d"}},
		{"starts with hyphen", "-hello", []string{"", "hello"}},
		{"ends with hyphen", "hello-", []string{"hello", ""}},
		{"just hyphens", "---", []string{"", "", "", ""}},
		{"empty", "", []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitOnHyphen(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("splitOnHyphen(%q) length = %d, want %d", tt.input, len(got), len(tt.expected))
			} else {
				for i := range got {
					if got[i] != tt.expected[i] {
						t.Errorf("splitOnHyphen(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
					}
				}
			}
		})
	}
}

func TestToUpper(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected rune
	}{
		{"lower a", 'a', 'A'},
		{"lower z", 'z', 'Z'},
		{"upper A", 'A', 'A'},
		{"upper Z", 'Z', 'Z'},
		{"number 5", '5', '5'},
		{"space", ' ', ' '},
		{"emoji", '😀', '😀'},
		{"non-ascii lower", 'é', 'é'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toUpper(tt.input)
			if got != tt.expected {
				t.Errorf("toUpper(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAvatarSizeClass(t *testing.T) {
	tests := []struct {
		name     string
		size     AvatarSize
		expected string
	}{
		{"sm", AvatarSm, "w-8 h-8 rounded-full flex-shrink-0"},
		{"md", AvatarMd, "w-10 h-10 rounded-full flex-shrink-0"},
		{"lg", AvatarLg, "w-10 h-10 rounded-full ring-2 ring-transparent group-hover:ring-july-500/50 transition-all"},
		{"default", AvatarSize("xlg"), "w-8 h-8 rounded-full flex-shrink-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AvatarSizeClass(tt.size)
			if got != tt.expected {
				t.Errorf("AvatarSizeClass(%q) = %q, want %q", tt.size, got, tt.expected)
			}
		})
	}
}

func TestAvatarCircleClass(t *testing.T) {
	tests := []struct {
		name     string
		size     AvatarSize
		expected string
	}{
		{"sm", AvatarSm, "w-8 h-8 rounded-full bg-july-500/20 flex items-center justify-center text-july-400 text-sm flex-shrink-0"},
		{"md", AvatarMd, "w-10 h-10 rounded-full bg-july-500/20 flex items-center justify-center text-july-400 text-sm flex-shrink-0"},
		{"lg", AvatarLg, "w-10 h-10 rounded-full bg-gradient-to-br from-july-500 to-purple-600 flex items-center justify-center text-white font-bold"},
		{"default", AvatarSize("xlg"), "w-8 h-8 rounded-full bg-july-500/20 flex items-center justify-center text-july-400 text-sm flex-shrink-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AvatarCircleClass(tt.size)
			if got != tt.expected {
				t.Errorf("AvatarCircleClass(%q) = %q, want %q", tt.size, got, tt.expected)
			}
		})
	}
}
