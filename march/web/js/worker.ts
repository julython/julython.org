/**
 * Analyzer Web Worker
 *
 * Fetches repo structure via the Julython GitHub proxy, caches file content
 * in an LRU map, scores the repo in pure TS, and posts results back.
 *
 * Message protocol:
 *   IN:  { type: "analyze", owner: string, repo: string }
 *   OUT: { type: "progress", message: string }
 *       | { type: "result",  scorecard: Scorecard }
 *       | { type: "error",   message: string }
 */

// ── Types ────────────────────────────────────────────────────────────────────

interface TreeEntry {
  path: string;
  type: "blob" | "tree";
  sha: string;
  size?: number;
}

interface RepoMeta {
  full_name: string;
  description: string | null;
  default_branch: string;
  stargazers_count: number;
  open_issues_count: number;
}

export interface ScoredCategory {
  name: string;
  score: number;   // 0–100
  max: number;
  signals: string[];
}

export interface Scorecard {
  repo: string;
  total: number;   // 0–100
  categories: ScoredCategory[];
}

// ── LRU cache ────────────────────────────────────────────────────────────────

const MAX_CACHE = 200;
const cache = new Map<string, string>();

function cacheGet(key: string): string | null {
  if (!cache.has(key)) return null;
  const val = cache.get(key)!;
  cache.delete(key);
  cache.set(key, val);
  return val;
}

function cacheSet(key: string, val: string) {
  cache.delete(key);
  cache.set(key, val);
  if (cache.size > MAX_CACHE) {
    const oldest = cache.keys().next().value;
    if (oldest !== undefined) cache.delete(oldest);
  }
}

// ── GitHub proxy ──────────────────────────────────────────────────────────────

const PROXY = "/api/v1/gh";

async function ghGet<T>(path: string, params?: Record<string, string>): Promise<T> {
  const url = new URL(`${PROXY}/${path}`, self.location.origin);
  if (params) for (const [k, v] of Object.entries(params)) url.searchParams.set(k, v);
  const res = await fetch(url.toString());
  if (!res.ok) throw new Error(`GitHub proxy ${res.status} for ${path}`);
  return res.json();
}

async function fetchContent(owner: string, repo: string, path: string, ref: string): Promise<string> {
  const key = `${owner}/${repo}/${path}@${ref}`;
  const hit = cacheGet(key);
  if (hit !== null) return hit;

  const data = await ghGet<{ content: string }>(`repos/${owner}/${repo}/contents/${path}`, { ref });
  const content = atob(data.content.replace(/\n/g, ""));
  cacheSet(key, content);
  return content;
}

// ── Tree walking ──────────────────────────────────────────────────────────────

// Dirs worth one level of expansion for structure signals.
const INTERESTING_DIRS = new Set([
  "src", "lib", "app", "pkg", "cmd",
  "tests", "test", "spec", "__tests__",
  "docs", "doc", ".github",
]);

// Files whose content we fetch for deeper scoring.
const FETCH_FILES = new Set([
  "README.md", "README.rst", "README.txt",
  "CONTRIBUTING.md", "CONTRIBUTING.rst",
  "CHANGELOG.md", "CHANGELOG.rst",
  "pyproject.toml", "setup.py", "setup.cfg",
  "Cargo.toml", "package.json", "go.mod",
  "Makefile", "justfile",
  ".pre-commit-config.yaml",
]);

async function collectStructure(owner: string, repo: string, ref: string) {
  const root = await ghGet<{ tree: TreeEntry[] }>(`repos/${owner}/${repo}/git/trees/${ref}`);

  const allPaths: string[] = [];
  const fetchPaths: string[] = [];

  for (const entry of root.tree) {
    if (entry.type === "blob") {
      allPaths.push(entry.path);
      if (FETCH_FILES.has(entry.path)) fetchPaths.push(entry.path);
    } else if (entry.type === "tree" && INTERESTING_DIRS.has(entry.path)) {
      const sub = await ghGet<{ tree: TreeEntry[] }>(`repos/${owner}/${repo}/git/trees/${entry.sha}`);
      for (const child of sub.tree) {
        if (child.type === "blob") allPaths.push(`${entry.path}/${child.path}`);
      }
    }
  }

  return { allPaths, fetchPaths };
}

// ── Scorer ────────────────────────────────────────────────────────────────────

function hasAny(paths: string[], patterns: RegExp[]): boolean {
  return paths.some(p => patterns.some(rx => rx.test(p)));
}

function score(repo: RepoMeta, paths: string[], contents: Record<string, string>): Scorecard {
  const categories: ScoredCategory[] = [];

  // ── Documentation (max 25) ───────────────────────────────────────────────
  {
    let s = 0;
    const signals: string[] = [];
    const readme = contents["README.md"] ?? contents["README.rst"] ?? contents["README.txt"] ?? "";

    if (readme.length > 500)  { s += 10; signals.push("README present and substantial"); }
    else if (readme.length)   { s += 4;  signals.push("README present but brief"); }

    if (readme.length > 2000) { s += 5;  signals.push("README is detailed (>2k chars)"); }

    if (contents["CONTRIBUTING.md"] || contents["CONTRIBUTING.rst"]) {
      s += 5; signals.push("CONTRIBUTING guide present");
    }
    if (contents["CHANGELOG.md"] || contents["CHANGELOG.rst"]) {
      s += 5; signals.push("Changelog present");
    }
    if (hasAny(paths, [/^docs?\//i])) { s += 5; signals.push("docs/ directory present"); }

    categories.push({ name: "Documentation", score: Math.min(s, 25), max: 25, signals });
  }

  // ── Testing (max 25) ─────────────────────────────────────────────────────
  {
    let s = 0;
    const signals: string[] = [];

    if (hasAny(paths, [/^tests?\//i, /^spec\//i, /__tests__\//i])) {
      s += 10; signals.push("Test directory present");
    }
    if (hasAny(paths, [/test.*\.(py|ts|js|go|rs)$/i])) {
      s += 5; signals.push("Test files found");
    }
    if (hasAny(paths, [/\.github\/workflows\//i])) {
      s += 10; signals.push("CI workflow present");
    }
    const pyproject = contents["pyproject.toml"] ?? "";
    if (pyproject.includes("[tool.pytest") || pyproject.includes("pytest")) {
      s += 5; signals.push("pytest configured");
    }

    categories.push({ name: "Testing", score: Math.min(s, 25), max: 25, signals });
  }

  // ── Project hygiene (max 25) ─────────────────────────────────────────────
  {
    let s = 0;
    const signals: string[] = [];

    if (hasAny(paths, [/^\.github\/ISSUE_TEMPLATE/i, /^\.github\/PULL_REQUEST_TEMPLATE/i])) {
      s += 5; signals.push("Issue/PR templates present");
    }
    if (hasAny(paths, [/^LICENSE/i, /^COPYING/i])) {
      s += 10; signals.push("License file present");
    }
    if (contents[".pre-commit-config.yaml"]) {
      s += 5; signals.push("pre-commit hooks configured");
    }
    if (hasAny(paths, [/^\.gitignore$/])) {
      s += 5; signals.push(".gitignore present");
    }
    if (repo.open_issues_count < 50) {
      s += 5; signals.push("Open issue count is healthy (<50)");
    }

    categories.push({ name: "Hygiene", score: Math.min(s, 25), max: 25, signals });
  }

  // ── Build / packaging (max 25) ───────────────────────────────────────────
  {
    let s = 0;
    const signals: string[] = [];
    const hasMake = !!contents["Makefile"] || !!contents["justfile"];

    if (contents["pyproject.toml"]) { s += 10; signals.push("pyproject.toml present"); }
    else if (contents["setup.py"])  { s += 5;  signals.push("setup.py present"); }
    if (contents["go.mod"])         { s += 10; signals.push("go.mod present"); }
    if (contents["Cargo.toml"])     { s += 10; signals.push("Cargo.toml present"); }
    if (contents["package.json"])   { s += 8;  signals.push("package.json present"); }
    if (hasMake)                    { s += 5;  signals.push("Makefile / justfile present"); }
    if (hasAny(paths, [/Dockerfile/i, /docker-compose/i])) {
      s += 5; signals.push("Docker config present");
    }

    categories.push({ name: "Build & Packaging", score: Math.min(s, 25), max: 25, signals });
  }

  const total = Math.round(
    categories.reduce((sum, c) => sum + (c.score / c.max) * 25, 0)
  );

  return { repo: repo.full_name, total, categories };
}

// ── Main ──────────────────────────────────────────────────────────────────────

async function analyze(owner: string, repo: string): Promise<Scorecard> {
  post("progress", `Fetching metadata for ${owner}/${repo}…`);
  const meta = await ghGet<RepoMeta>(`repos/${owner}/${repo}`);
  const ref = meta.default_branch;

  post("progress", `Walking tree at ${ref}…`);
  const { allPaths, fetchPaths } = await collectStructure(owner, repo, ref);

  post("progress", `Fetching ${fetchPaths.length} scored files…`);
  const contents: Record<string, string> = {};
  await Promise.all(
    fetchPaths.map(async path => {
      try { contents[path] = await fetchContent(owner, repo, path, ref); }
      catch { /* absent file = absent signal, scorer handles it */ }
    })
  );

  post("progress", "Scoring…");
  return score(meta, allPaths, contents);
}

self.addEventListener("message", async (e: MessageEvent) => {
  const { type, owner, repo } = e.data;
  if (type !== "analyze") return;
  try {
    const scorecard = await analyze(owner, repo);
    self.postMessage({ type: "result", scorecard });
  } catch (err) {
    self.postMessage({ type: "error", message: (err as Error).message });
  }
});

function post(type: string, message: string) {
  self.postMessage({ type, message });
}