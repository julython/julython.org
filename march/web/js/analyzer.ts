/**
 * Project Analyzer — main thread
 *
 * Mounted by the project detail templ page via:
 *   <div id="analyzer" data-repo-url="https://github.com/owner/repo"
 *                      data-worker-url="/assets/worker.abc123.js"></div>
 *   <script type="module" src="/assets/analyzer.abc123.js"></script>
 *
 * Spawns the worker, streams progress, and renders the scorecard + chat UI.
 */

import type { Scorecard, ScoredCategory } from "./worker";

// ── Bootstrap ─────────────────────────────────────────────────────────────────

const mount = document.getElementById("analyzer");
if (mount) {
  const repoURL = mount.dataset.repoUrl ?? "";
  const workerURL = mount.dataset.workerUrl ?? "";
  if (repoURL && workerURL) {
    mount.appendChild(buildUI(repoURL, workerURL));
  }
}

// ── UI builder ────────────────────────────────────────────────────────────────

function buildUI(repoURL: string, workerURL: string): HTMLElement {
  const { owner, repo } = parseRepoURL(repoURL);

  const root = el("div", "mt-8 border border-white/10 rounded-xl bg-surface-light overflow-hidden");

  // Header bar
  const header = el("div", "flex items-center justify-between px-5 py-4 border-b border-white/10");
  const title = el("h2", "text-sm font-semibold text-white");
  title.textContent = "Repo Analyzer";
  const btn = el("button",
    "px-4 py-1.5 rounded-lg text-sm font-medium bg-july-500/20 border border-july-500/40 " +
    "text-july-300 hover:bg-july-500/30 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
  ) as HTMLButtonElement;
  btn.textContent = "Analyze & Report";
  header.append(title, btn);

  // Progress / results area
  const body = el("div", "px-5 py-4 space-y-4");

  // Chat area (hidden until analysis complete)
  const chatWrap = el("div", "hidden border-t border-white/10 pt-4 mt-2");
  const chatLog = el("div", "space-y-2 mb-3 max-h-48 overflow-y-auto text-sm text-gray-300");
  const chatRow = el("div", "flex gap-2");
  const chatInput = el("input",
    "flex-1 px-3 py-2 rounded-lg bg-surface border border-white/10 text-white text-sm " +
    "placeholder-gray-500 focus:outline-none focus:border-july-500/50"
  ) as HTMLInputElement;
  chatInput.placeholder = "Ask about this repo…";
  const chatBtn = el("button",
    "px-3 py-2 rounded-lg bg-july-500/20 border border-july-500/40 text-july-300 text-sm " +
    "hover:bg-july-500/30 transition-colors"
  ) as HTMLButtonElement;
  chatBtn.textContent = "Ask";
  chatRow.append(chatInput, chatBtn);
  chatWrap.append(chatLog, chatRow);

  root.append(header, body, chatWrap);

  // ── Wire up button ──────────────────────────────────────────────────────────
  let scorecard: Scorecard | null = null;
  let repoFiles: Record<string, string> = {};

  btn.addEventListener("click", () => {
    btn.disabled = true;
    btn.textContent = "Analyzing…";
    body.innerHTML = "";
    chatWrap.classList.add("hidden");
    scorecard = null;

    const worker = new Worker(workerURL, { type: "module" });

    worker.addEventListener("message", (e: MessageEvent) => {
      const { type, message, scorecard: sc } = e.data;

      if (type === "progress") {
        showProgress(body, message);
      } else if (type === "result") {
        scorecard = sc as Scorecard;
        repoFiles = e.data.fileContents ?? {};
        body.innerHTML = "";
        body.appendChild(renderScorecard(scorecard));
        chatWrap.classList.remove("hidden");
        btn.textContent = "Re-analyze";
        btn.disabled = false;
        worker.terminate();
      } else if (type === "error") {
        showError(body, message);
        btn.textContent = "Retry";
        btn.disabled = false;
        worker.terminate();
      }
    });

    worker.postMessage({ type: "analyze", owner, repo });
  });

  // ── Chat handler ────────────────────────────────────────────────────────────
  const sendChat = () => {
    const q = chatInput.value.trim();
    if (!q || !scorecard) return;
    chatInput.value = "";
    appendChat(chatLog, "you", q);
    appendChat(chatLog, "ai", chat(q, scorecard, repoFiles));
  };

  chatBtn.addEventListener("click", sendChat);
  chatInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendChat(); }
  });

  return root;
}

// ── Scorecard renderer ────────────────────────────────────────────────────────

function renderScorecard(sc: Scorecard): HTMLElement {
  const wrap = el("div", "space-y-4");

  // Total score badge
  const totalRow = el("div", "flex items-center gap-4");
  const badge = el("div", "text-4xl font-bold tabular-nums " + scoreColor(sc.total));
  badge.textContent = String(sc.total);
  const badgeSub = el("div", "text-xs text-gray-500 leading-tight");
  badgeSub.innerHTML = "out of 100<br>overall score";
  totalRow.append(badge, badgeSub);
  wrap.appendChild(totalRow);

  // Category bars
  for (const cat of sc.categories) {
    wrap.appendChild(renderCategory(cat));
  }

  return wrap;
}

function renderCategory(cat: ScoredCategory): HTMLElement {
  const pct = Math.round((cat.score / cat.max) * 100);
  const wrap = el("div", "");

  const label = el("div", "flex justify-between text-xs mb-1");
  const name = el("span", "text-gray-300 font-medium");
  name.textContent = cat.name;
  const pts = el("span", "text-gray-500");
  pts.textContent = `${cat.score} / ${cat.max}`;
  label.append(name, pts);

  const track = el("div", "h-1.5 rounded-full bg-white/10 overflow-hidden");
  const bar = el("div", "h-full rounded-full transition-all " + barColor(pct));
  bar.style.width = `${pct}%`;
  track.appendChild(bar);

  const signals = el("ul", "mt-1.5 space-y-0.5");
  for (const s of cat.signals) {
    const li = el("li", "text-xs text-gray-500 flex items-center gap-1.5");
    li.innerHTML = `<span class="text-green-500">✓</span> ${s}`;
    signals.appendChild(li);
  }

  wrap.append(label, track, signals);
  return wrap;
}

// ── Chat (simple local LLM hook — swap fetch target as needed) ───────────────

function chat(question: string, sc: Scorecard, files: Record<string, string>): string {
  // For now: a rule-based responder using the scorecard data.
  // Replace this body with a fetch to /api/v1/chat or window.ai when ready.
  const low = sc.categories.filter(c => (c.score / c.max) < 0.5);
  if (/improve|suggest|fix|better/i.test(question)) {
    if (low.length === 0) return "This repo scores well across all categories — nice work!";
    return `Focus on: ${low.map(c => c.name).join(", ")}. ` +
      low.flatMap(c => c.signals.length === 0
        ? [`${c.name} has no detected signals — check the category requirements.`]
        : []
      ).join(" ");
  }
  if (/score|total/i.test(question)) {
    return `Overall score is ${sc.total}/100. ` +
      sc.categories.map(c => `${c.name}: ${c.score}/${c.max}`).join(", ") + ".";
  }
  return `I can see the repo scored ${sc.total}/100. Try asking "what should I improve?" or "explain the score".`;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function showProgress(container: HTMLElement, msg: string) {
  const p = el("p", "text-sm text-gray-400 flex items-center gap-2");
  p.innerHTML = `<span class="inline-block w-1.5 h-1.5 rounded-full bg-july-400 animate-pulse"></span>${msg}`;
  container.appendChild(p);
}

function showError(container: HTMLElement, msg: string) {
  container.innerHTML = "";
  const p = el("p", "text-sm text-red-400");
  p.textContent = `Error: ${msg}`;
  container.appendChild(p);
}

function appendChat(log: HTMLElement, who: "you" | "ai", text: string) {
  const row = el("div", who === "you" ? "text-right" : "text-left");
  const bubble = el("span",
    who === "you"
      ? "inline-block px-3 py-1.5 rounded-lg bg-july-500/20 text-july-200 text-sm"
      : "inline-block px-3 py-1.5 rounded-lg bg-white/5 text-gray-300 text-sm"
  );
  bubble.textContent = text;
  row.appendChild(bubble);
  log.appendChild(row);
  log.scrollTop = log.scrollHeight;
}

function el(tag: string, cls: string): HTMLElement {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  return e;
}

function scoreColor(n: number): string {
  if (n >= 75) return "text-green-400";
  if (n >= 50) return "text-july-400";
  return "text-red-400";
}

function barColor(pct: number): string {
  if (pct >= 75) return "bg-green-500";
  if (pct >= 50) return "bg-july-500";
  return "bg-red-500";
}

function parseRepoURL(url: string): { owner: string; repo: string } {
  const clean = url.replace(/^https?:\/\/github\.com\//, "").replace(/\.git$/, "").trim();
  const [owner, repo] = clean.split("/");
  if (!owner || !repo) throw new Error(`Cannot parse repo from: ${url}`);
  return { owner, repo };
}