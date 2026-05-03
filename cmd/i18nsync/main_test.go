package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeyToLabel(t *testing.T) {
	tests := []struct {
		in, out string
	}{
			{"Home", "home"},
			{"SignIn", "sign in"},
			{"ShowProjects", "show projects"},
			{"Error404Title", "error title"},
			{"Profile", "profile"},
			{"a", "a"},
			{"getHTTPResponse", "h t t p response"},
			{"profile.overview", "overview"},
			{"projects.allServices", "services"},
			{"NoProjects", "no projects"},
			{"LoadingMore", "loading more"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got := keyToLabel(tc.in)
			if got != tc.out {
				t.Errorf("%q = %q, want %q", tc.in, got, tc.out)
				}
			})
		}
}

func TestSortedNonEmpty(t *testing.T) {
	m := map[string][]missingEntry{
			"ns2": {{key: "b"}},
			"":     {{key: "a"}, {key: "d"}},
			"ns1": {{key: "c"}},
	}
	got := sortedNonEmpty(m)
	want := []string{"ns1", "ns2"}
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d: %v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("sortedNonEmpty[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveValue_withTranslation(t *testing.T) {
	translations := map[string]string{"Home": "Inicio", "SignIn": "Iniciar sesion"}

	// Direct match
	e := missingEntry{key: "Home"}
	if got := resolveValue(e, translations, false); got != "Inicio" {
		t.Errorf("resolveValue = %q, want %q", got, "Inicio")
	}

	// Case-insensitive match (LLMs lowercase keys)
	e = missingEntry{key: "signin"}
	if got := resolveValue(e, translations, false); got != "Iniciar sesion" {
		t.Errorf("resolveValue (case-insensitive) = %q, want %q", got, "Iniciar sesion")
	}
}

func TestResolveValue_fallback(t *testing.T) {
	translations := map[string]string{"Home": "Inicio"}

	// Key not in translations with allowFallback=true
	e := missingEntry{key: "NewKey"}
	if got := resolveValue(e, translations, true); got != "new key" {
		t.Errorf("with allowFallback=true = %q, want %q", got, "new key")
	}
}

func TestResolveValue_fullKey(t *testing.T) {
	// LLMs often return the full namespaced key even when the local key was requested.
	translations := map[string]string{"profile.webhooksSubtitle": "Webhooks conectados"}

	e := missingEntry{key: "webhooksSubtitle", fullKey: "profile.webhooksSubtitle"}
	if got := resolveValue(e, translations, false); got != "Webhooks conectados" {
		t.Errorf("resolveValue(fullKey) = %q, want %q", got, "Webhooks conectados")
	}

	// When fullKey is empty, local key should still work.
	e2 := missingEntry{key: "Home", fullKey: ""}
	translations2 := map[string]string{"Home": "Inicio"}
	if got := resolveValue(e2, translations2, false); got != "Inicio" {
		t.Errorf("resolveValue(empty fullKey) = %q, want %q", got, "Inicio")
	}
}

func TestParseLocaleRaw(t *testing.T) {
	yaml := `es:
  Home: "Inicio"
  About: "Acerca de"
  profile:
    overview: "Resumen"
    settings: "Configuración"
  CommitCount:
    one: "%{count} commit"
    other: "%{count} commits"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "es.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := parseLocaleRaw(path)
	if err != nil {
		t.Fatalf("parseLocaleRaw: %v", err)
	}

	// Check singular keys
	if _, ok := result.singular["Home"]; !ok {
		t.Error("expected singular key 'Home'")
	}
	if _, ok := result.singular["About"]; !ok {
		t.Error("expected singular key 'About'")
	}
	if _, ok := result.singular["profile"]; !ok {
		t.Error("expected singular key 'profile'")
	}
	if _, ok := result.singular["profile.overview"]; !ok {
		t.Error("expected singular key 'profile.overview'")
	}
	if _, ok := result.singular["profile.settings"]; !ok {
		t.Error("expected singular key 'profile.settings'")
	}

	// Check plural keys
	if _, ok := result.plural["CommitCount"]; !ok {
		t.Error("expected plural key 'CommitCount'")
	}
}

func TestParseLocaleRaw_notFound(t *testing.T) {
	_, err := parseLocaleRaw("/no/such/file.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseLocaleRaw_emptyLocale(t *testing.T) {
	yaml := `fr:
  Title: "Bonjour"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "fr.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := parseLocaleRaw(path)
	if err != nil {
		t.Fatalf("parseLocaleRaw: %v", err)
	}
	if result.root == nil {
		t.Fatal("expected root map to be non-nil")
	}
	if _, ok := result.root["Title"]; !ok {
		t.Error("expected root key 'Title'")
	}
}

func TestParseLocaleRaw_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml: [[["), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := parseLocaleRaw(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestSyncFile_dryRun(t *testing.T) {
	enYAML := `en:
  Home: "Home"
  About: "About"
  profile:
    title: "Title"
`
	esYAML := `es:
  Home: "Inicio"
`
	enDir := t.TempDir()
	esDir := t.TempDir()
	enPath := filepath.Join(enDir, "en.yaml")
	esPath := filepath.Join(esDir, "es.yaml")

	if err := os.WriteFile(enPath, []byte(enYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(esPath, []byte(esYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	source, err := parseLocaleRaw(enPath)
	if err != nil {
		t.Fatalf("parseLocaleRaw(en): %v", err)
	}

	keys := keySet{
		singular: map[string]struct{}{
			"Home":         {},
			"About":        {},
			"profile.title": {},
		},
		plural: map[string]struct{}{},
	}

	// Dry run should not modify the file.
	if err := syncFile(esPath, source, keys, true, false); err != nil {
		t.Fatalf("syncFile dry-run: %v", err)
	}

	// File should be unchanged.
	data, err := os.ReadFile(esPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != esYAML {
		t.Errorf("file was modified in dry-run:\ngot:\n%s\nwant:\n%s", data, esYAML)
	}
}

func TestSyncFile_writesNewKeys(t *testing.T) {
	t.Setenv("OLLAMA_API_URL", "http://localhost:1")
	enYAML := `en:
  Home: "Home"
  About: "About"
  Help:
    one: "%{count} help topic"
    other: "%{count} help topics"
`
	esYAML := `es:
  Home: "Inicio"
`
	enDir := t.TempDir()
	esDir := t.TempDir()
	enPath := filepath.Join(enDir, "en.yaml")
	esPath := filepath.Join(esDir, "es.yaml")

	if err := os.WriteFile(enPath, []byte(enYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(esPath, []byte(esYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	source, err := parseLocaleRaw(enPath)
	if err != nil {
		t.Fatalf("parseLocaleRaw(en): %v", err)
	}

	keys := keySet{
		singular: map[string]struct{}{
			"Home":      {},
			"About":     {},
		},
		plural: map[string]struct{}{
			"Help": {},
		},
	}

	if err := syncFile(esPath, source, keys, false, true); err != nil {
		t.Fatalf("syncFile: %v", err)
	}

	data, err := os.ReadFile(esPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if !strings.Contains(got, "About") {
		t.Errorf("synced file missing 'About':\n%s", got)
	}
	if !strings.Contains(got, "Home") {
		t.Errorf("synced file missing 'Home':\n%s", got)
	}
	if !strings.Contains(got, "Help") {
		t.Errorf("synced file missing plural 'Help':\n%s", got)
	}
	if !strings.Contains(got, "one") {
		t.Errorf("synced file missing plural 'one':\n%s", got)
	}
	if !strings.Contains(got, "other") {
		t.Errorf("synced file missing plural 'other':\n%s", got)
	}
}

func TestSyncFile_upToDate(t *testing.T) {
	enYAML := `en:
  Home: "Home"
`
	esYAML := `es:
  Home: "Inicio"
`
	enDir := t.TempDir()
	esDir := t.TempDir()
	enPath := filepath.Join(enDir, "en.yaml")
	esPath := filepath.Join(esDir, "es.yaml")

	if err := os.WriteFile(enPath, []byte(enYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(esPath, []byte(esYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	source, err := parseLocaleRaw(enPath)
	if err != nil {
		t.Fatalf("parseLocaleRaw(en): %v", err)
	}

	keys := keySet{
		singular: map[string]struct{}{"Home": {}},
		plural:   map[string]struct{}{},
	}

	if err := syncFile(esPath, source, keys, false, true); err != nil {
		t.Fatalf("syncFile: %v", err)
	}

	data, err := os.ReadFile(esPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != esYAML {
		t.Errorf("file should not change when up to date:\ngot:\n%s\nwant:\n%s", data, esYAML)
	}
}

func TestSyncFile_namespaceGrouping(t *testing.T) {
	// Point Ollama at unreachable port to ensure keyToLabel fallback
	t.Setenv("OLLAMA_API_URL", "http://localhost:1")

	enYAML := `en:
  profile:
    title: "Title"
    name: "Name"
  Home: "Home"
  About: "About"
`
	esYAML := `es:
  Home: "Inicio"
`
	enDir := t.TempDir()
	esDir := t.TempDir()
	enPath := filepath.Join(enDir, "en.yaml")
	esPath := filepath.Join(esDir, "es.yaml")

	if err := os.WriteFile(enPath, []byte(enYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(esPath, []byte(esYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	source, err := parseLocaleRaw(enPath)
	if err != nil {
		t.Fatalf("parseLocaleRaw(en): %v", err)
	}

	keys := keySet{
		singular: map[string]struct{}{
			"profile.title":   {},
			"profile.name":    {},
			"Home":            {},
			"About":           {},
		},
		plural: map[string]struct{}{},
	}

	if err := syncFile(esPath, source, keys, false, true); err != nil {
		t.Fatalf("syncFile: %v", err)
	}

	data, err := os.ReadFile(esPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "profile:") {
		t.Error("synced file missing 'profile:' namespace block")
	}
	if !strings.Contains(got, "title:") {
		t.Error("synced file missing 'title:'")
	}
	if !strings.Contains(got, "name:") {
		t.Error("synced file missing 'name:'")
	}
}

func TestSyncFile_pluralWithMultiLineSource(t *testing.T) {
	t.Setenv("OLLAMA_API_URL", "http://localhost:1")
	enYAML := `en:
  HelpTopics:
    one: "%{count} topic"
    other: "%{count} topics"
  LongText: >-
    This is a long text that may contain
    multiple lines and various characters.
`
	esYAML := `es:
  Home: "Inicio"
`
	enDir := t.TempDir()
	esDir := t.TempDir()
	enPath := filepath.Join(enDir, "en.yaml")
	esPath := filepath.Join(esDir, "es.yaml")

	if err := os.WriteFile(enPath, []byte(enYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(esPath, []byte(esYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	source, err := parseLocaleRaw(enPath)
	if err != nil {
		t.Fatalf("parseLocaleRaw(en): %v", err)
	}

	keys := keySet{
		singular: map[string]struct{}{
			"LongText":  {},
		},
		plural: map[string]struct{}{
			"HelpTopics": {},
		},
	}

	if err := syncFile(esPath, source, keys, false, true); err != nil {
		t.Fatalf("syncFile: %v", err)
	}

	data, err := os.ReadFile(esPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "LongText") {
		t.Error("synced file missing 'LongText'")
	}
	if !strings.Contains(got, "HelpTopics") {
		t.Error("synced file missing 'HelpTopics' plural")
	}
}

func TestScanFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	code := `package main

import "july/internal/i18n"

func handler(ctx context.Context) {
	_ = i18n.T(ctx, "Home")
	_ = i18n.T(ctx, "SignIn.SignIn")
	_ = i18n.N(ctx, "ExtractCount", "one", "c1", "c2")
	_ = i18n.T(ctx, "About")
}
`
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	ks := newKeySet()
	if err := scanFile(path, ks); err != nil {
		t.Fatalf("scanFile: %v", err)
	}

	// Should have 3 singular keys
	if len(ks.singular) != 3 {
		t.Errorf("singular keys = %d, want 3: keys = %v", len(ks.singular), ks.singular)
	}
	if _, ok := ks.singular["Home"]; !ok {
		t.Error("missing singular key 'Home'")
	}
	if _, ok := ks.singular["SignIn.SignIn"]; !ok {
		t.Error("missing singular key 'SignIn.SignIn'")
	}
	if _, ok := ks.singular["About"]; !ok {
		t.Error("missing singular key 'About'")
	}
	// Should have 1 plural key
	if len(ks.plural) != 1 {
		t.Errorf("plural keys = %d, want 1: keys = %v", len(ks.plural), ks.plural)
	}
	if _, ok := ks.plural["ExtractCount"]; !ok {
		t.Error("missing plural key 'ExtractCount'")
	}
}

func TestScanFile_skipsTemplGenerated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file_templ.go")
	if err := os.WriteFile(path, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	ks := newKeySet()
	if err := scanFile(path, ks); err != nil {
		t.Fatalf("scanFile: %v", err)
	}
	if len(ks.singular) != 0 {
		t.Errorf("expected no keys from _templ.go file, got %d", len(ks.singular))
	}
}

func TestParseLocaleRaw_values(t *testing.T) {
	yamlContent := `en:
  Home: "Home"
  About: "About"
  profile:
    title: "Title"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "en.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := parseLocaleRaw(path)
	if err != nil {
		t.Fatalf("parseLocaleRaw: %v", err)
	}

	if result.root["Home"] != "Home" {
		t.Errorf("Home = %q, want %q", result.root["Home"], "Home")
	}
	if result.root["About"] != "About" {
		t.Errorf("About = %q, want %q", result.root["About"], "About")
	}
	if result.root["profile"] != nil {
		// profile is a map[string]any, check its sub-keys
		profileMap, ok := result.root["profile"].(map[string]any)
		if !ok {
			t.Fatal("profile should be a map")
			}
		if profileMap["title"] != "Title" {
				t.Errorf("profile.title = %q, want %q", profileMap["title"], "Title")
			}
		}
}

func TestExtractKeys(t *testing.T) {
	dir := t.TempDir()

	// Create a Go file with i18n keys.
	goFile := filepath.Join(dir, "handler.go")
	goCode := `package main

import "julython/internal/i18n"

func f(ctx context.Context) {
	_ = i18n.T(ctx, "ExtractKey1")
	_ = i18n.N(ctx, "ExtractCount", "one", "c1", "c2")
}
`
	if err := os.WriteFile(goFile, []byte(goCode), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a templ file with i18n keys.
	templFile := filepath.Join(dir, "view.templ")
	templCode := `package mypackage

import "julython/internal/i18n"

func v(ctx context.Context) {
	templ.Text(i18n.T(ctx, "ExtractKey2"))
}
`
	if err := os.WriteFile(templFile, []byte(templCode), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a .yml file (should be skipped).
	ymlFile := filepath.Join(dir, "skip.yml")
	if err := os.WriteFile(ymlFile, []byte("skip: yes"), 0o644); err != nil {
		t.Fatal(err)
	}

	ks, err := extractKeys(dir)
	if err != nil {
		t.Fatalf("extractKeys: %v", err)
	}

	// Should have 2 singular keys + 1 plural key
	if len(ks.singular) != 2 {
		t.Errorf("singular = %d, want 2: %v", len(ks.singular), ks.singular)
	}
	if len(ks.plural) != 1 {
		t.Errorf("plural = %d, want 1: %v", len(ks.plural), ks.plural)
	}
	if _, ok := ks.singular["ExtractKey1"]; !ok {
		t.Error("missing singular key 'ExtractKey1'")
	}
	if _, ok := ks.singular["ExtractKey2"]; !ok {
		t.Error("missing singular key 'ExtractKey2'")
	}
	if _, ok := ks.plural["ExtractCount"]; !ok {
		t.Error("missing plural key 'ExtractCount'")
	}
}
