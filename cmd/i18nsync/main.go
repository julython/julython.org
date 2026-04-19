// i18nsync walks Go and templ source files, extracts every i18n.T and i18n.N
// key, and appends missing entries to locale YAML files.
//
// i18n.T keys are written as-is (flat or namespaced).
// i18n.N keys are written with one/other plural structure.
//
// Usage:
//
//	go run ./cmd/i18nsync
//	go run ./cmd/i18nsync -src . -locales locales/ -dry-run
//
//go:generate go run ./cmd/i18nsync
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	tKeyRe = regexp.MustCompile(`i18n\.T\([^,]+,\s*"([^"]+)"`)
	nKeyRe = regexp.MustCompile(`i18n\.N\([^,]+,\s*"([^"]+)"`)
)

type keySet struct {
	singular map[string]struct{} // i18n.T keys
	plural   map[string]struct{} // i18n.N keys
}

func newKeySet() keySet {
	return keySet{
		singular: map[string]struct{}{},
		plural:   map[string]struct{}{},
	}
}

func main() {
	srcDir := flag.String("src", ".", "root directory to scan")
	locDir := flag.String("locales", "internal/i18n/locales", "directory containing locale YAML files")
	dryRun := flag.Bool("dry-run", false, "print changes without writing files")
	flag.Parse()

	keys, err := extractKeys(*srcDir)
	if err != nil {
		fatalf("scan error: %v\n", err)
	}
	fmt.Printf("found %d singular and %d plural keys\n", len(keys.singular), len(keys.plural))

	locales, _ := filepath.Glob(filepath.Join(*locDir, "*.yaml"))
	if len(locales) == 0 {
		locales, _ = filepath.Glob(filepath.Join(*locDir, "*.yml"))
	}
	if len(locales) == 0 {
		fatalf("no YAML files found in %s\n", *locDir)
	}

	for _, path := range locales {
		if err := syncFile(path, keys, *dryRun); err != nil {
			fatalf("error syncing %s: %v\n", path, err)
		}
	}
}

func extractKeys(root string) (keySet, error) {
	ks := newKeySet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name != "." && (strings.HasPrefix(name, ".") || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		switch {
		case ext == ".templ":
			// scan templ source files
		case ext == ".go" && !strings.HasSuffix(path, "_templ.go"):
			// scan Go files, but skip templ-generated output
		default:
			return nil
		}
		return scanFile(path, ks)
	})
	return ks, err
}

func scanFile(path string, ks keySet) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, m := range tKeyRe.FindAllStringSubmatch(line, -1) {
			ks.singular[m[1]] = struct{}{}
		}
		for _, m := range nKeyRe.FindAllStringSubmatch(line, -1) {
			ks.plural[m[1]] = struct{}{}
		}
	}
	return scanner.Err()
}

// ============================================
// YAML sync
// ============================================

type existingKeys struct {
	singular map[string]struct{} // flat or "ns.Key"
	plural   map[string]struct{} // keys with one/other children
}

func syncFile(path string, keys keySet, dryRun bool) error {
	existing, err := loadExistingKeys(path)
	if err != nil {
		return err
	}

	var missingSingular, missingPlural []string
	for k := range keys.singular {
		if _, ok := existing.singular[k]; !ok {
			missingSingular = append(missingSingular, k)
		}
	}
	for k := range keys.plural {
		if _, ok := existing.plural[k]; !ok {
			missingPlural = append(missingPlural, k)
		}
	}
	sort.Strings(missingSingular)
	sort.Strings(missingPlural)

	total := len(missingSingular) + len(missingPlural)
	if total == 0 {
		fmt.Printf("%s: up to date\n", path)
		return nil
	}

	fmt.Printf("%s: adding %d missing keys:\n", path, total)
	for _, k := range missingSingular {
		fmt.Printf("  + %s\n", k)
	}
	for _, k := range missingPlural {
		fmt.Printf("  + %s (plural)\n", k)
	}

	if dryRun {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	fmt.Fprintln(w, "")

	// Singular: group by namespace, flat keys first.
	grouped := map[string][]string{}
	for _, k := range missingSingular {
		ns, _, hasNs := strings.Cut(k, ".")
		if hasNs {
			grouped[ns] = append(grouped[ns], k)
		} else {
			grouped[""] = append(grouped[""], k)
		}
	}
	if flat := grouped[""]; len(flat) > 0 {
		for _, k := range flat {
			fmt.Fprintf(w, "  %s: \"%s\"\n", k, keyToLabel(k))
		}
	}
	nsList := sortedNonEmpty(grouped)
	for _, ns := range nsList {
		fmt.Fprintf(w, "  %s:\n", ns)
		for _, k := range grouped[ns] {
			_, key, _ := strings.Cut(k, ".")
			fmt.Fprintf(w, "    %s: \"%s\"\n", key, keyToLabel(key))
		}
	}

	// Plural: always flat for now (namespaced plurals are uncommon).
	for _, k := range missingPlural {
		label := keyToLabel(k)
		fmt.Fprintf(w, "  %s:\n", k)
		fmt.Fprintf(w, "    one: \"%%{count} %s\"\n", label)
		fmt.Fprintf(w, "    other: \"%%{count} %ss\"\n", label)
	}

	return w.Flush()
}

func loadExistingKeys(path string) (existingKeys, error) {
	f, err := os.Open(path)
	if err != nil {
		return existingKeys{}, err
	}
	defer f.Close()

	result := existingKeys{
		singular: map[string]struct{}{},
		plural:   map[string]struct{}{},
	}

	lineRe := regexp.MustCompile(`^(\s*)([A-Za-z][A-Za-z0-9_]*):\s*`)
	pluralSubkeys := map[string]bool{"one": true, "other": true, "zero": true, "two": true, "few": true, "many": true}

	var currentBlock string // name of the current 2-space block
	var blockIsPlural bool

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indent, name := len(m[1]), m[2]
		trimmed := strings.TrimSpace(line)

		switch indent {
		case 0: // locale root — skip
		case 2:
			isBlock := strings.HasSuffix(trimmed, ":")
			if isBlock {
				currentBlock = name
				blockIsPlural = false // determined by first child
			} else {
				currentBlock = ""
				blockIsPlural = false
				result.singular[name] = struct{}{}
			}
		case 4:
			if currentBlock == "" {
				continue
			}
			if pluralSubkeys[name] {
				// This block is a plural key.
				blockIsPlural = true
				result.plural[currentBlock] = struct{}{}
			} else if !blockIsPlural {
				// This block is a namespace.
				result.singular[currentBlock+"."+name] = struct{}{}
			}
		}
	}
	return result, scanner.Err()
}

// ============================================
// Helpers
// ============================================

var splitUpperRe = regexp.MustCompile(`[A-Z][a-z]*`)

// keyToLabel converts a CamelCase key to a lowercase label.
// "ShowProjects" → "show projects"
func keyToLabel(key string) string {
	// Use only the local part if namespaced.
	_, local, hasNs := strings.Cut(key, ".")
	if hasNs {
		key = local
	}
	words := splitUpperRe.FindAllString(key, -1)
	if len(words) == 0 {
		return strings.ToLower(key)
	}
	return strings.ToLower(strings.Join(words, " "))
}

func sortedNonEmpty(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
