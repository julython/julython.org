// assetgen runs Tailwind CSS + esbuild, hashes outputs, writes to web/assets/,
// and generates internal/components/assets_gen.go with filenames as constants.
//
// @mlc-ai/web-llm is vendored at web/vendor/mlc-web-llm/ by make setup (gitignored).
// The npm package version is pinned only in the Makefile for the download URL.
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
)

const (
	tailwindBin       = "bin/tailwindcss"
	inputCSS          = "web/input.css"
	assetsDir         = "web/assets"
	manifestPath      = "internal/components/assets_gen.go"
	webLLMVendorIndex = "web/vendor/mlc-web-llm/lib/index.js"
)

func main() {
	dev := flag.Bool("dev", false, "dev mode: no hash, no minify")
	flag.Parse()

	webllmAlias, err := webLLMIndexAbs()
	if err != nil {
		fatalf("web-llm vendor: %v\n", err)
	}

	cssName, err := buildCSS(*dev)
	if err != nil {
		fatalf("css build: %v\n", err)
	}

	htmxName, err := findVendored("htmx-*.min.js")
	if err != nil {
		fatalf("htmx: %v\n", err)
	}

	mermaidName, err := findVendored("mermaid-*.min.js")
	if err != nil {
		fatalf("mermaid: %v\n", err)
	}

	analyzerName, err := buildJS("web/js/analyzer.ts", "analyzer", *dev, webllmAlias)
	if err != nil {
		fatalf("analyzer build: %v\n", err)
	}

	llmWorkerName, err := buildJS("web/js/llm-worker.ts", "llm-worker", *dev, webllmAlias)
	if err != nil {
		fatalf("llm-worker build: %v\n", err)
	}

	if err := writeManifest(cssName, htmxName, mermaidName, analyzerName, llmWorkerName); err != nil {
		fatalf("manifest: %v\n", err)
	}

	fmt.Printf("assets_gen.go → tailwind: %s  htmx: %s  mermaid: %s  analyzer: %s  llm-worker: %s\n",
		cssName, htmxName, mermaidName, analyzerName, llmWorkerName)
}

// webLLMIndexAbs returns an absolute path to vendored lib/index.js for esbuild Alias.
func webLLMIndexAbs() (string, error) {
	if _, err := os.Stat(webLLMVendorIndex); err != nil {
		return "", fmt.Errorf("%q missing — run make setup from the march directory: %w", webLLMVendorIndex, err)
	}
	return filepath.Abs(webLLMVendorIndex)
}

func buildCSS(dev bool) (string, error) {
	if err := os.MkdirAll("tmp", 0o755); err != nil {
		return "", err
	}
	tmp := "tmp/tailwind-out.css"

	args := []string{"-i", inputCSS, "-o", tmp}
	if !dev {
		args = append(args, "--minify")
	}
	cmd := exec.Command(tailwindBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tailwindcss: %w", err)
	}

	data, err := os.ReadFile(tmp)
	if err != nil {
		return "", err
	}

	return writeHashed(data, "tailwind", ".css", dev)
}

// buildJS bundles a TypeScript entry point with esbuild and writes the result
// to web/assets/ with a content hash. Returns the hashed filename.
func buildJS(entry, stem string, dev bool, webLLMAlias string) (string, error) {
	minify := !dev
	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{entry},
		Bundle:            true,
		MinifyWhitespace:  minify,
		MinifyIdentifiers: minify,
		MinifySyntax:      minify,
		Format:            api.FormatESModule,
		Target:            api.ES2020,
		Platform:          api.PlatformBrowser,
		External:          []string{"node:*", "url", "path", "fs", "crypto"},
		Alias: map[string]string{
			"@mlc-ai/web-llm": webLLMAlias,
		},
		// Stub out process so the typeof window === 'undefined' branch
		// in @mlc-ai/web-llm resolves to the browser path at runtime.
		Banner: map[string]string{
			"js": "const process = undefined;",
		},
		Write:    false,
		LogLevel: api.LogLevelWarning,
	})

	if len(result.Errors) > 0 {
		msg := result.Errors[0].Text
		if result.Errors[0].Location != nil {
			msg = fmt.Sprintf("%s:%d: %s", result.Errors[0].Location.File,
				result.Errors[0].Location.Line, msg)
		}
		return "", fmt.Errorf("esbuild: %s", msg)
	}

	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("esbuild produced no output for %s", entry)
	}

	return writeHashed(result.OutputFiles[0].Contents, stem, ".js", dev)
}

// writeHashed writes data to web/assets/{stem}.{hash}{ext} and removes stale
// versions. In dev mode the filename is just {stem}{ext} with no hash.
func writeHashed(data []byte, stem, ext string, dev bool) (string, error) {
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return "", err
	}

	var name string
	if dev {
		name = stem + ext
	} else {
		hash := fmt.Sprintf("%x", sha256.Sum256(data))[:8]
		name = fmt.Sprintf("%s.%s%s", stem, hash, ext)

		// Remove stale hashed versions.
		old, _ := filepath.Glob(filepath.Join(assetsDir, stem+".????????"+ext))
		for _, f := range old {
			if filepath.Base(f) != name {
				_ = os.Remove(f)
			}
		}
	}

	dest := filepath.Join(assetsDir, name)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", err
	}
	return name, nil
}

func findVendored(pattern string) (string, error) {
	matches, _ := filepath.Glob(filepath.Join(assetsDir, pattern))
	if len(matches) == 0 {
		return "", fmt.Errorf("%s not found in %s — run: make setup", pattern, assetsDir)
	}
	return filepath.Base(matches[0]), nil
}

func writeManifest(cssName, htmxName, mermaidName, analyzerName, llmWorkerName string) error {
	content := fmt.Sprintf(`// Code generated by assetgen — DO NOT EDIT.
package components

const (
	AssetTailwindCSS = "/assets/%s"
	AssetHTMX        = "/assets/%s"
	AssetMermaid     = "/assets/%s"
	AssetAnalyzer    = "/assets/%s"
	AssetLLMWorker   = "/assets/%s"
)
`, cssName, htmxName, mermaidName, analyzerName, llmWorkerName)
	return os.WriteFile(manifestPath, []byte(content), 0o644)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "assetgen: "+format, args...)
	os.Exit(1)
}
