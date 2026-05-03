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
//	go run ./cmd/i18nsync -allow-fallback   # allow keyToLabel fallback on Ollama failure
//
//go:generate go run ./cmd/i18nsync
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	tKeyRe = regexp.MustCompile(`i18n\.T\(\s*ctx\s*,\s*"([^"]+)"`)
	nKeyRe = regexp.MustCompile(`i18n\.N\(\s*ctx\s*,\s*"([^"]+)"`)
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

// missingEntry holds the English text for each missing key.
type missingEntry struct {
	key     string // local key (e.g. "webhooksSubtitle")
	fullKey string // full namespaced key (e.g. "profile.webhooksSubtitle")
	value   string
}

// localeRaw holds the parsed root map and classified keys from a locale YAML file.
type localeRaw struct {
	code   string              // e.g. "es", "en"
	root   map[string]any // the inner map under the locale key
	singular map[string]struct{}
	plural   map[string]struct{}
}

func main() {
	srcDir := flag.String("src", ".", "root directory to scan")
	locDir := flag.String("locales", "internal/i18n/locales", "directory containing locale YAML files")
	dryRun := flag.Bool("dry-run", false, "print changes without writing files")
	allowFallback := flag.Bool("allow-fallback", false, "allow falling back to keyToLabel on Ollama failure")
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

	// Determine the source locale path (en.yaml).
	sourcePath := filepath.Join(*locDir, "en.yaml")
	sourceLocale, err := parseLocaleRaw(sourcePath)
	if err != nil {
		logf("warning: could not load source locale %s: %v", sourcePath, err)
		sourceLocale = &localeRaw{root: make(map[string]any), singular: map[string]struct{}{}}
	}

	for _, path := range locales {
		if err := syncFile(path, sourceLocale, keys, *dryRun, *allowFallback); err != nil {
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
		case ext == ".go" && !strings.HasSuffix(path, "_templ.go") && !strings.HasSuffix(path, "_test.go"):
			// scan Go files, but skip templ-generated and test files
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

var pluralSubkeys = map[string]bool{
	"one": true, "other": true, "zero": true,
	"two": true, "few": true, "many": true,
}

// coerceYAML converts map[interface{}]any values from yaml.v3 into map[string]any
// so we can work with string keys.
func coerceYAML(v any) any {
	switch val := v.(type) {
	case map[interface{}]any:
		m := make(map[string]any, len(val))
		for k, v := range val {
			m[fmt.Sprintf("%v", k)] = coerceYAML(v)
		}
		return m
	case []any:
		out := make([]any, len(val))
		for i, v := range val {
			out[i] = coerceYAML(v)
		}
		return out
	default:
		return v
	}
}

// parseLocaleRaw reads a locale YAML file and returns the inner map (for merging)
// along with the classified singular/plural keys.
func parseLocaleRaw(path string) (*localeRaw, error) {
	data := &localeRaw{
		root:     make(map[string]any),
		singular: make(map[string]struct{}),
		plural:   make(map[string]struct{}),
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return data, err
	}

	var raw any
	if err := yaml.Unmarshal(bytes, &raw); err != nil {
		return data, err
	}

	// The YAML root is a map with one key: the locale code (e.g. "es", "pt").
	m, ok := raw.(map[string]any)
	if !ok {
		return data, fmt.Errorf("unexpected YAML structure in %s", path)
	}

	// Get the locale root (should be the only key).
	for code, v := range m {
		localeRoot := coerceYAML(v)
		root, ok := localeRoot.(map[string]any)
		if !ok {
			return data, fmt.Errorf("unexpected locale root type in %s", path)
		}
		data.code = code
		data.root = root
		traverseNode(root, "", pluralSubkeys, data.singular, data.plural)
	}

	return data, nil
}

// getNestedValue looks up a dotted key like "profile.title" in a nested map structure.
func getNestedValue(root map[string]any, key string) any {
	parts := strings.Split(key, ".")
	node := root
	for i, p := range parts {
		val, ok := node[p]
		if !ok {
			return nil
		}
		if i == len(parts)-1 {
			return val
		}
		if next, ok := val.(map[string]any); ok {
			node = next
		} else {
			return nil
		}
	}
	return nil
}

// traverseNode recursively processes a YAML map node, classifying keys as singular or plural.
func traverseNode(node map[string]any, prefix string, pluralSubkeys map[string]bool, singular, plural map[string]struct{}) {
	for k, v := range node {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		switch val := v.(type) {
		case string:
			singular[fullKey] = struct{}{}
		case map[string]any:
			// Check if this block is plural (has plural sub-keys).
			isPlural := false
			for subKey := range val {
				if pluralSubkeys[subKey] {
					isPlural = true
					break
				}
			}
			if isPlural {
				plural[fullKey] = struct{}{}
			} else {
				singular[fullKey] = struct{}{}
				traverseNode(val, fullKey, pluralSubkeys, singular, plural)
			}
		}
	}
}

func syncFile(path string, sourceLocale *localeRaw, keys keySet, dryRun, allowFallback bool) error {
	existing, err := parseLocaleRaw(path)
	if err != nil {
		return err
	}

	// Collect missing entries with their English source values.
	var missingEntries []missingEntry
	var missingSingular, missingPlural []string
	for k := range keys.singular {
		if _, ok := existing.singular[k]; !ok {
			missingSingular = append(missingSingular, k)
			if srcVal := getNestedValue(sourceLocale.root, k); srcVal != nil {
				srcStr, _ := srcVal.(string)
				missingEntries = append(missingEntries, missingEntry{
					key:     k,
					fullKey: k,
					value:   srcStr,
				})
			}
		}
	}
	for k := range keys.plural {
		if _, ok := existing.plural[k]; !ok {
			missingPlural = append(missingPlural, k)
			if srcVal := getNestedValue(sourceLocale.root, k); srcVal != nil {
				srcStr, _ := srcVal.(string)
				missingEntries = append(missingEntries, missingEntry{
					key:     k,
					fullKey: k,
					value:   srcStr,
				})
			}
		}
	}
	sort.Strings(missingSingular)
	sort.Strings(missingPlural)

	total := len(missingSingular) + len(missingPlural)
	if total == 0 {
		fmt.Printf("%s: up to date\n", path)
		return nil
	}

	// Determine target language from filename.
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	targetLang := strings.ToLower(base) // "es", "pt", etc.
	isEnglish := targetLang == "en"

	// Try Ollama translation for non-English locales.
	var translations map[string]string
	if !isEnglish && !dryRun {
		client := NewOllamaClient()
		if isOllamaAvailable(context.Background(), client.BaseURL) {
			translations, err = client.Translate(context.Background(), missingEntries, targetLang)
			if err != nil {
				if !allowFallback {
					return fmt.Errorf("Ollama translation failed: %w", err)
				}
				logf("warning: Ollama translation failed for %s: %v — falling back to keyToLabel", path, err)
				translations = nil
			}
		} else {
			if !allowFallback {
				return fmt.Errorf("Ollama not available at %s", client.BaseURL)
			}
			logf("warning: Ollama not available at %s — falling back to keyToLabel for %s", client.BaseURL, path)
		}
	}

	fmt.Printf("%s: adding %d missing keys:\n", path, total)
	for _, k := range missingSingular {
		fmt.Printf("     + %s\n", k)
	}
	for _, k := range missingPlural {
		fmt.Printf("     + %s (plural)\n", k)
	}

	if dryRun {
		return nil
	}

	// Singular keys: group by namespace, write flat keys then grouped.
	singularGrouped := map[string][]missingEntry{}
	for _, e := range missingEntries {
		if strings.Contains(e.key, ".") {
	ns := strings.Split(e.key, ".")[0]
			_, local, _ := strings.Cut(e.key, ".")
		e.key = local
			singularGrouped[ns] = append(singularGrouped[ns], e)
		} else {
			singularGrouped[""] = append(singularGrouped[""], e)
		}
	}

	// Write flat keys.
	for _, e := range singularGrouped[""] {
		existing.root[e.key] = resolveValue(e, translations, allowFallback)
	}
	for _, ns := range sortedNonEmpty(singularGrouped) {
		nsMap := existing.root[ns]
		if nsMap == nil {
			nsMap = make(map[string]any)
			existing.root[ns] = nsMap
		}
		if nsRoot, ok := nsMap.(map[string]any); ok {
			for _, e := range singularGrouped[ns] {
				nsRoot[e.key] = resolveValue(e, translations, allowFallback)
			}
		}
	}

	// Plural: always flat for now (namespaced plurals are uncommon).
	for _, k := range missingPlural {
		label := keyToLabel(k)
		existing.root[k] = map[string]string{
			"one":   fmt.Sprintf("%%{count} %s", label),
			"other": fmt.Sprintf("%%{count} %ss", label),
		}
	}

	merged, err := yaml.Marshal(map[string]any{existing.code: existing.root})
	if err != nil {
		return fmt.Errorf("marshal merged locale: %w", err)
	}

	return os.WriteFile(path, merged, 0o644)
}

// resolveValue returns the translated value for an entry, falling back to
// keyToLabel if no translation is available.
func resolveValue(e missingEntry, translations map[string]string, allowFallback bool) string {
	if translations != nil {
			// Try local key first, then full namespaced key (LLMs may return either).
		for _, key := range func() []string {
				if e.fullKey != "" && e.fullKey != e.key {
					return []string{e.key, e.fullKey}
				}
				return []string{e.key}
			}() {
			if t, ok := translations[key]; ok {
				return t
				}
				// Case-insensitive lookup (LLMs often lowercase keys).
			for k, t := range translations {
				if strings.EqualFold(k, key) {
					return t
					}
				}
			}
		if !allowFallback {
			logf("error: key %q not in Ollama response and --allow-fallback not set — aborting", e.key)
			os.Exit(1)
		}
		logf("warning: key %q not in Ollama response — falling back to keyToLabel", e.key)
	}
	return keyToLabel(e.key)
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

func sortedNonEmpty(m map[string][]missingEntry) []string {
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
