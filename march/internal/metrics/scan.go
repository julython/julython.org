package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

// ErrGitHubForbidden is returned when GitHub responds 403 (e.g. private repo or token cannot read).
// Callers may treat this as "mark project private" for mis-labeled public rows; note 403 can also mean rate/abuse limits.
var ErrGitHubForbidden = errors.New("github: forbidden")

const graphqlEndpoint = "https://api.github.com/graphql"

type TreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	Size int    `json:"size"`
}

type FileMatch struct {
	Path     string
	Category string
	Content  string // populated in pass 2
}

type L1ScanResult struct {
	Owner      string
	Repo       string
	SHA        string
	Language   string // from GitHub's primaryLanguage (Linguist)
	Tree       []TreeEntry
	Matches    []FileMatch
	ByCategory map[string][]FileMatch
}

type Client struct {
	httpClient *http.Client
	token      string
}

func NewClient(token string) *Client {
	return &Client{httpClient: &http.Client{}, token: token}
}

// FetchL1Scan runs two GraphQL queries:
// 1. Fetch the full recursive tree to discover what exists
// 2. Fetch content for files relevant to L1 scoring
func (c *Client) FetchL1Scan(ctx context.Context, owner, repo string) (*L1ScanResult, error) {
	sha, language, tree, err := c.fetchTree(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("pass 1 tree fetch: %w", err)
	}

	matches := classifyTree(tree)
	if len(matches) == 0 {
		return &L1ScanResult{
			Owner:      owner,
			Repo:       repo,
			SHA:        sha,
			Language:   language,
			Tree:       tree,
			Matches:    matches,
			ByCategory: map[string][]FileMatch{},
		}, nil
	}

	paths := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, m := range matches {
		if !seen[m.Path] {
			paths = append(paths, m.Path)
			seen[m.Path] = true
		}
	}

	files, err := c.fetchFilesBatched(ctx, owner, repo, sha, paths)
	if err != nil {
		return nil, fmt.Errorf("pass 2 file fetch: %w", err)
	}

	byCategory := make(map[string][]FileMatch)
	for i := range matches {
		matches[i].Content = files[matches[i].Path]
		byCategory[matches[i].Category] = append(
			byCategory[matches[i].Category], matches[i],
		)
	}

	return &L1ScanResult{
		Owner:      owner,
		Repo:       repo,
		SHA:        sha,
		Language:   language,
		Tree:       tree,
		Matches:    matches,
		ByCategory: byCategory,
	}, nil
}

// fetchTree gets the full recursive tree via the Git Trees API through GraphQL.
func (c *Client) fetchTree(ctx context.Context, owner, repo string) (string, string, []TreeEntry, error) {
	query := fmt.Sprintf(`query {
  repository(owner: %q, name: %q) {
    primaryLanguage { name }
    defaultBranchRef {
      target {
        oid
        ... on Commit {
          tree { oid }
        }
      }
    }
  }
}`, owner, repo)

	resp, err := c.doGraphQL(ctx, query)
	if err != nil {
		return "", "", nil, err
	}

	repoFields, err := extractRepo(resp)
	if err != nil {
		return "", "", nil, err
	}

	var language string
	if raw, ok := repoFields["primaryLanguage"]; ok && string(raw) != "null" {
		var pl struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &pl); err == nil {
			language = pl.Name
		}
	}

	var branch struct {
		Target struct {
			OID  string `json:"oid"`
			Tree struct {
				OID string `json:"oid"`
			} `json:"tree"`
		} `json:"target"`
	}
	if err := json.Unmarshal(repoFields["defaultBranchRef"], &branch); err != nil {
		return "", "", nil, fmt.Errorf("unmarshal branch: %w", err)
	}

	tree, truncated, err := c.fetchRecursiveTree(ctx, owner, repo, branch.Target.Tree.OID)
	if err != nil {
		return "", "", nil, err
	}
	if truncated {
		return "", "", nil, fmt.Errorf("git tree response truncated (repo too large)")
	}

	return branch.Target.OID, language, tree, nil
}

const graphqlFileBatchSize = 28

// fetchRecursiveTree calls the REST Git Trees API with recursive=1.
func (c *Client) fetchRecursiveTree(ctx context.Context, owner, repo, treeSHA string) ([]TreeEntry, bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, treeSHA)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusForbidden {
			return nil, false, fmt.Errorf("tree API: %w", ErrGitHubForbidden)
		}
		return nil, false, fmt.Errorf("tree API responded %d: %s", resp.StatusCode, b)
	}

	var result struct {
		SHA       string `json:"sha"`
		Truncated bool   `json:"truncated"`
		Tree      []struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
			Type string `json:"type"`
			Size int    `json:"size"`
			SHA  string `json:"sha"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, err
	}

	entries := make([]TreeEntry, 0, len(result.Tree))
	for _, e := range result.Tree {
		entries = append(entries, TreeEntry{
			Path: e.Path,
			Type: e.Type,
			Size: e.Size,
		})
	}
	return entries, result.Truncated, nil
}

// tile category signals — maps category to path matchers
type pathMatcher struct {
	Category string
	Match    func(path string, isBlob bool) bool
	// max files to fetch per category to keep the query bounded
	Limit int
}

var matchers = []pathMatcher{
	{
		Category: "readme",
		Limit:    1,
		Match: func(p string, isBlob bool) bool {
			base := strings.ToLower(path.Base(p))
			return isBlob && path.Dir(p) == "." &&
				strings.HasPrefix(base, "readme")
		},
	},
	{
		Category: "license",
		Limit:    1,
		Match: func(p string, isBlob bool) bool {
			base := strings.ToLower(path.Base(p))
			return isBlob && path.Dir(p) == "." &&
				(strings.HasPrefix(base, "license") || strings.HasPrefix(base, "licence"))
		},
	},
	{
		Category: "ci",
		Limit:    3,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			return strings.HasPrefix(p, ".github/workflows/") ||
				p == ".gitlab-ci.yml" ||
				strings.HasPrefix(p, ".circleci/") ||
				p == "Jenkinsfile" ||
				strings.HasPrefix(p, ".buildkite/")
		},
	},
	{
		Category: "tests",
		Limit:    5,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			base := path.Base(p)
			dir := strings.ToLower(path.Dir(p))
			// go test files
			if strings.HasSuffix(base, "_test.go") {
				return true
			}
			// python test files
			if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
				return true
			}
			// common test directories
			parts := strings.Split(dir, "/")
			for _, part := range parts {
				switch part {
				case "test", "tests", "__tests__", "spec", "specs":
					return true
				}
			}
			return false
		},
	},
	{
		Category: "linting",
		Limit:    3,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			base := path.Base(p)
			lintFiles := []string{
				".golangci.yml", ".golangci.yaml",
				".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs", ".eslintrc.yml",
				"eslint.config.js", "eslint.config.mjs",
				".flake8", "ruff.toml", ".ruff.toml",
				".prettierrc", ".prettierrc.json",
				".editorconfig", ".clang-format",
				"biome.json", "deno.json",
			}
			for _, f := range lintFiles {
				if base == f {
					return true
				}
			}
			return false
		},
	},
	{
		Category: "dependencies",
		Limit:    2,
		Match: func(p string, isBlob bool) bool {
			if !isBlob || path.Dir(p) != "." {
				return false
			}
			depFiles := []string{
				"go.mod", "go.sum",
				"package.json",
				"requirements.txt", "pyproject.toml", "setup.py", "setup.cfg",
				"Cargo.toml", "Gemfile", "pom.xml", "build.gradle",
				"composer.json", "mix.exs",
			}
			for _, f := range depFiles {
				if p == f {
					return true
				}
			}
			return false
		},
	},
	{
		Category: "docs",
		Limit:    3,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			dir := strings.Split(p, "/")[0]
			switch strings.ToLower(dir) {
			case "docs", "doc", "documentation", "wiki":
				return true
			}
			// mkdocs, sphinx, etc.
			base := path.Base(p)
			return base == "mkdocs.yml" || base == "conf.py" ||
				base == "docusaurus.config.js" || base == ".readthedocs.yml"
		},
	},
	{
		Category: "ai_readability",
		Limit:    3,
		Match: func(p string, isBlob bool) bool {
			if !isBlob {
				return false
			}
			base := strings.ToLower(path.Base(p))
			return base == "claude.md" || base == "agents.md" ||
				base == ".cursorrules" || base == ".cursorignore" ||
				base == "copilot-instructions.md" ||
				base == ".github/copilot-instructions.md" ||
				base == "coderabbit.yaml" || base == ".coderabbit.yaml"
		},
	},
}

// classifyTree walks the tree and picks out files worth fetching,
// returning them tagged with their tile category.
func classifyTree(tree []TreeEntry) []FileMatch {
	counts := make(map[string]int)
	var matches []FileMatch

	for _, entry := range tree {
		isBlob := entry.Type == "blob"
		for _, m := range matchers {
			if counts[m.Category] >= m.Limit {
				continue
			}
			if m.Match(entry.Path, isBlob) {
				matches = append(matches, FileMatch{
					Path:     entry.Path,
					Category: m.Category,
				})
				counts[m.Category]++
			}
		}
	}
	return matches
}

// fetchFilesBatched runs fetchFiles in chunks to stay under GitHub GraphQL complexity limits.
func (c *Client) fetchFilesBatched(ctx context.Context, owner, repo, sha string, paths []string) (map[string]string, error) {
	out := make(map[string]string, len(paths))
	for i := 0; i < len(paths); i += graphqlFileBatchSize {
		end := i + graphqlFileBatchSize
		if end > len(paths) {
			end = len(paths)
		}
		batch, err := c.fetchFiles(ctx, owner, repo, sha, paths[i:end])
		if err != nil {
			return nil, err
		}
		for k, v := range batch {
			out[k] = v
		}
	}
	return out, nil
}

// fetchFiles builds a GraphQL query with one alias per file path
// and returns a map of path -> content.
func (c *Client) fetchFiles(ctx context.Context, owner, repo, sha string, paths []string) (map[string]string, error) {
	if len(paths) == 0 {
		return map[string]string{}, nil
	}
	var b strings.Builder
	b.WriteString("query {\n")
	b.WriteString(fmt.Sprintf("  repository(owner: %q, name: %q) {\n", owner, repo))

	aliasToPath := make(map[string]string, len(paths))
	for i, p := range paths {
		alias := fmt.Sprintf("f%d", i)
		aliasToPath[alias] = p
		expr := fmt.Sprintf("%s:%s", sha, p)
		b.WriteString(fmt.Sprintf("    %s: object(expression: %q) {\n", alias, expr))
		b.WriteString("      ... on Blob { text byteSize }\n")
		b.WriteString("    }\n")
	}

	b.WriteString("  }\n}\n")

	resp, err := c.doGraphQL(ctx, b.String())
	if err != nil {
		return nil, err
	}

	repoFields, err := extractRepo(resp)
	if err != nil {
		return nil, err
	}

	files := make(map[string]string, len(paths))
	for alias, filePath := range aliasToPath {
		raw, ok := repoFields[alias]
		if !ok || string(raw) == "null" {
			continue
		}
		var blob struct {
			Text     *string `json:"text"`
			ByteSize int     `json:"byteSize"`
		}
		if err := json.Unmarshal(raw, &blob); err == nil && blob.Text != nil {
			files[filePath] = *blob.Text
		}
	}

	return files, nil
}

// --- helpers ---

func (c *Client) doGraphQL(ctx context.Context, query string) (map[string]json.RawMessage, error) {
	body, err := json.Marshal(graphqlRequest{Query: query})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("github graphql: %w", ErrGitHubForbidden)
		}
		return nil, fmt.Errorf("github responded %d: %s", resp.StatusCode, b)
	}

	var raw struct {
		Data   map[string]json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if len(raw.Errors) > 0 {
		return nil, fmt.Errorf("graphql: %s", raw.Errors[0].Message)
	}

	return raw.Data, nil
}

func extractRepo(data map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	repoRaw, ok := data["repository"]
	if !ok {
		return nil, fmt.Errorf("no repository in response")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(repoRaw, &fields); err != nil {
		return nil, fmt.Errorf("unmarshal repository: %w", err)
	}
	return fields, nil
}

type graphqlRequest struct {
	Query string `json:"query"`
}
