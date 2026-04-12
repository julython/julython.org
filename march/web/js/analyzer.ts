// Server-backed analysis (L1) + browser LLM for metric reviews and general questions.
import {
  createSession, deleteCachedModels,
  getCachedModels, MODELS, DEFAULT_MODEL,
  type LLMSession,
} from "./llm";

const LLM_MODEL_STORAGE_KEY = "july-llm-model";

// ── Bootstrap ─────────────────────────────────────────────────────────────────

const mount = document.getElementById("analyzer");
if (mount) {
  const llmWorkerURL = mount.dataset.llmWorkerUrl;
  const projectID = mount.dataset.projectId?.trim() ?? "";
  const repoURL = mount.dataset.repoUrl?.trim() ?? "";
  const repoName = mount.dataset.repoName?.trim() ?? "";
  const repoDescription = mount.dataset.repoDescription?.trim() ?? "";
  if (llmWorkerURL && projectID && repoURL) {
    mount.appendChild(buildUI({
      repoURL,
      repoName: repoName || repoURL,
      repoDescription,
      projectID,
      llmWorkerURL,
    }));
  }
}

const METRIC_LABELS: Record<string, string> = {
  readme: "README",
  tests: "Tests",
  ci: "CI",
  structure: "Structure",
  linting: "Linting",
  deps: "Dependencies",
  docs: "Documentation",
  ai_ready: "AI-ready",
};

type UIConfig = {
  repoURL: string;
  repoName: string;
  repoDescription: string;
  projectID: string;
  llmWorkerURL: string;
};

/** Model chooser lives in the Assistant panel header (#llm-model-select). */
function getModelSelect(): HTMLSelectElement {
  const el = document.getElementById("llm-model-select") as HTMLSelectElement | null;
  if (!el) {
    throw new Error("missing #llm-model-select (expected in Assistant header)");
  }
  if (el.options.length === 0) {
    for (const m of MODELS) {
      const opt = document.createElement("option");
      opt.value = m.id;
      opt.textContent = m.label;
      el.appendChild(opt);
    }
    const saved = localStorage.getItem(LLM_MODEL_STORAGE_KEY);
    if (saved && MODELS.some(m => m.id === saved)) {
      el.value = saved;
    } else {
      el.value = DEFAULT_MODEL;
    }
  }
  return el;
}

function getCacheClearButton(): HTMLButtonElement | null {
  return document.getElementById("llm-cache-clear") as HTMLButtonElement | null;
}

function buildUI(cfg: UIConfig): HTMLElement {
  const root = el("div", "flex flex-col flex-1 min-h-[280px]");

  const modelSelect = getModelSelect();
  const deleteBtn = getCacheClearButton();

  const chatLog = el("div", "flex-1 min-h-[160px] max-h-80 overflow-y-auto px-4 py-3 space-y-2 text-sm sm:text-base text-gray-300");
  const chatRow = el("div", "flex gap-2 px-4 pb-4 shrink-0");
  const chatInput = el("input",
    "flex-1 px-3 py-2 rounded-lg bg-surface border border-white/10 text-white text-sm " +
    "placeholder-gray-500 focus:outline-none focus:border-july-500/50"
  ) as HTMLInputElement;
  chatInput.placeholder = "Ask about this repository…";
  const chatBtn = el("button",
    "px-3 py-2 rounded-lg bg-july-500/20 border border-july-500/40 text-july-300 text-sm " +
    "hover:bg-july-500/30 transition-colors disabled:opacity-40 disabled:cursor-not-allowed shrink-0"
  ) as HTMLButtonElement;
  chatBtn.textContent = "Send";
  chatRow.append(chatInput, chatBtn);

  root.append(chatLog, chatRow);

  let llmSession: LLMSession | null = null;

  function appendBrowserAiIntro() {
    appendChat(
      chatLog,
      "ai",
      "Runs in your browser. The selected model downloads once and is cached for next time.",
    );
  }
  appendBrowserAiIntro();

  const refreshDeleteBtn = async () => {
    if (!deleteBtn) return;
    if (!deleteBtn.dataset.defaultLabel) {
      deleteBtn.dataset.defaultLabel = deleteBtn.textContent.trim() || "Clear";
    }
    const hint = (deleteBtn.dataset.clearHint ?? "").trim();
    const cached = await getCachedModels();
    const hit = cached.find((c) => c.modelId === modelSelect.value);
    deleteBtn.textContent = deleteBtn.dataset.defaultLabel;
    if (hit) {
      deleteBtn.title = hint ? `${hint} (${hit.sizeMB} MB on disk)` : `${hit.sizeMB} MB on disk`;
    } else {
      deleteBtn.title = hint;
    }
  };
  void refreshDeleteBtn();
  modelSelect.addEventListener("change", () => {
    localStorage.setItem(LLM_MODEL_STORAGE_KEY, modelSelect.value);
    llmSession?.destroy();
    llmSession = null;
    void refreshDeleteBtn();
  });

  deleteBtn?.addEventListener("click", async () => {
    const modelId = modelSelect.value;
    const rec = MODELS.find(m => m.id === modelId)!;
    if (!confirm(`Delete the cached "${rec.label}" model? You'll need to re-download it next time.`)) return;
    deleteBtn.disabled = true;
    deleteBtn.textContent = "Deleting…";
    await deleteCachedModels(modelId);
    await refreshDeleteBtn();
    deleteBtn.disabled = false;
  });

  async function getOrCreateLlmSession(systemPrompt: string): Promise<LLMSession | null> {
    if (llmSession) {
      llmSession.destroy();
      llmSession = null;
    }
    const modelId = modelSelect.value;
    const statusBubble = appendChat(chatLog, "ai", "");
    setSpinner(statusBubble, "Checking for browser AI…");
    try {
      llmSession = await createSession(
        systemPrompt,
        modelId,
        cfg.llmWorkerURL,
        (msg) => { statusBubble.textContent = msg; },
      );
      statusBubble.closest("div")?.remove();
      await refreshDeleteBtn();
      return llmSession;
    } catch (e) {
      statusBubble.textContent = (e as Error).message;
      return null;
    }
  }

  function destroyLlmSession() {
    llmSession?.destroy();
    llmSession = null;
  }

  const sendChat = async () => {
    const q = chatInput.value.trim();
    if (!q) return;
    chatInput.value = "";
    chatBtn.disabled = true;
    appendChat(chatLog, "you", q);

    let bundle: { systemPrompt: string; userPrompt: string };
    try {
      const res = await fetch(
        `/api/projects/${encodeURIComponent(cfg.projectID)}/analysis/chat-context`,
        {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ message: q }),
        },
      );
      if (!res.ok) {
        const raw = await res.text().catch(() => res.statusText);
        appendChat(chatLog, "ai", formatMetricLLMHttpError(res.status, raw));
        chatBtn.disabled = false;
        return;
      }
      bundle = await res.json() as { systemPrompt: string; userPrompt: string };
    } catch (e) {
      appendChat(chatLog, "ai", `**Error:** ${(e as Error).message}`);
      chatBtn.disabled = false;
      return;
    }

    destroyLlmSession();

    const modelId = modelSelect.value;
    const statusBubble = appendChat(chatLog, "ai", "");
    setSpinner(statusBubble, "Checking for browser AI…");
    try {
      llmSession = await createSession(
        bundle.systemPrompt,
        modelId,
        cfg.llmWorkerURL,
        (msg) => { statusBubble.textContent = msg; },
      );
      statusBubble.closest("div")?.remove();
      await refreshDeleteBtn();
    } catch (e) {
      statusBubble.textContent = (e as Error).message;
      chatBtn.disabled = false;
      return;
    }

    const bubble = appendChat(chatLog, "ai", "");
    setSpinner(bubble, "Thinking…");
    await llmSession!.prompt(bundle.userPrompt, (fullText) => { updateBubble(bubble, fullText); });
    chatBtn.disabled = false;
  };

  chatBtn.addEventListener("click", sendChat);
  chatInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); void sendChat(); }
  });

  function scrollToAssistantPanel() {
    document.getElementById("project-assistant-panel")?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  }

  if (cfg.projectID) {
    document.addEventListener("click", (ev) => {
      const t = ev.target as HTMLElement | null;
      const tile = t?.closest?.(".metric-ai-tile") as HTMLButtonElement | null;
      if (!tile || !document.body.contains(tile)) return;
      scrollToAssistantPanel();
      if (mount?.dataset.metricAiEnabled !== "true") return;
      void runMetricAIReview(tile.dataset.metricKey ?? "", cfg.projectID, {
        chatLog,
        getOrCreateLlmSession,
        destroyLlmSession,
      });
    });
  }

  return root;
}

type MetricAIAdapter = {
  chatLog: HTMLElement;
  getOrCreateLlmSession: (systemPrompt: string) => Promise<LLMSession | null>;
  destroyLlmSession: () => void;
};

async function runMetricAIReview(
  metricKey: string,
  projectID: string,
  a: MetricAIAdapter,
) {
  if (!metricKey) return;

  const label = METRIC_LABELS[metricKey] ?? metricKey;
  appendChat(a.chatLog, "you", `AI review: ${label}`);

  const url = `/api/projects/${encodeURIComponent(projectID)}/analysis/metrics/${encodeURIComponent(metricKey)}/llm-context`;
  let res: Response;
  try {
    res = await fetch(url, { credentials: "same-origin" });
  } catch (e) {
    appendChat(a.chatLog, "ai", `**Error:** ${(e as Error).message}`);
    return;
  }
  if (!res.ok) {
    const raw = await res.text().catch(() => res.statusText);
    appendChat(a.chatLog, "ai", formatMetricLLMHttpError(res.status, raw));
    return;
  }
  const ctx = await res.json() as {
    repoName: string;
    systemPrompt: string;
    userPrompt: string;
  };

  const userPrompt = typeof ctx.userPrompt === "string" ? ctx.userPrompt.trim() : "";
  if (!userPrompt) {
    appendChat(a.chatLog, "ai", "**Error:** Empty analysis context from the server.");
    return;
  }

  const session = await a.getOrCreateLlmSession(ctx.systemPrompt);
  if (!session) return;

  const bubble = appendChat(a.chatLog, "ai", "");
  setSpinner(bubble, "Analyzing metric…");
  let raw = "";
  try {
    raw = await session.prompt(userPrompt, (full) => { updateBubble(bubble, full); });
  } catch (e) {
    bubble.textContent = `Error: ${(e as Error).message}`;
    a.destroyLlmSession();
    return;
  }

  updateBubble(
    bubble,
    raw.trim()
      ? `**${label}**\n\n${raw.trim()}`
      : "(empty response)",
  );

  a.destroyLlmSession();
}

// ── Helpers ─────────────────────────────────────────────────────────────────

/** Parses JSON error bodies from GET …/llm-context; falls back to plain text. */
function formatMetricLLMHttpError(status: number, raw: string): string {
  try {
    const j = JSON.parse(raw) as {
      error?: string;
      message?: string;
      metricHelpUrl?: string;
      helpUrl?: string;
    };
    if (typeof j.message === "string" && j.message.trim()) {
      let msg = j.message.trim();
      const link = j.metricHelpUrl || j.helpUrl;
      if (link) {
        msg += `\n\n[Learn more](${link})`;
      }
      return msg;
    }
  } catch {
    /* plain text body */
  }
  return `**Error (${status}):** ${raw || "unknown"}`;
}

function setSpinner(bubble: HTMLElement, label: string) {
  bubble.innerHTML =
    `<span class="inline-flex items-center gap-1.5 text-gray-500">` +
    `<span class="inline-flex gap-0.5">` +
    `<span class="w-1 h-1 rounded-full bg-gray-500 animate-bounce [animation-delay:-0.3s]"></span>` +
    `<span class="w-1 h-1 rounded-full bg-gray-500 animate-bounce [animation-delay:-0.15s]"></span>` +
    `<span class="w-1 h-1 rounded-full bg-gray-500 animate-bounce"></span>` +
    `</span>${label}</span>`;
}

function appendChat(log: HTMLElement, who: "you" | "ai", text: string): HTMLElement {
  const row = el("div", who === "you" ? "text-right" : "text-left");
  const bubble = el("span",
    who === "you"
      ? "inline-block px-3 py-1.5 rounded-lg bg-july-500/20 text-july-200 text-sm"
      : "inline-block px-3 py-1.5 rounded-lg bg-white/5 text-gray-300 text-sm text-left"
  );
  if (who === "ai") {
    bubble.innerHTML = renderMarkdown(text);
  } else {
    bubble.textContent = text;
  }
  row.appendChild(bubble);
  log.appendChild(row);
  log.scrollTop = log.scrollHeight;
  return bubble;
}

function updateBubble(bubble: HTMLElement, fullText: string) {
  bubble.innerHTML = renderMarkdown(fullText);
  const log = bubble.closest(".space-y-2") as HTMLElement | null;
  if (log) log.scrollTop = log.scrollHeight;
}

function renderMarkdown(text: string): string {
  if (!text) return "";
  return text
    .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
    .replace(/```[\w]*\n?([\s\S]*?)```/g, "<pre class='mt-1 p-2 rounded bg-black/30 text-xs overflow-x-auto whitespace-pre'>$1</pre>")
    .replace(/`([^`]+)`/g, "<code class='px-1 rounded bg-black/30 text-xs'>$1</code>")
    .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>")
    .replace(/\*(.*?)\*/g, "<em>$1</em>")
    .replace(/^#{1,3} (.+)$/gm, "<strong class='block mt-2'>$1</strong>")
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_m, label: string, href: string) => {
      const u = String(href).trim();
      const safe = (/^\//.test(u) || /^https:\/\//.test(u)) ? u.replace(/"/g, "") : "#";
      return `<a href="${safe}" class="text-july-400 hover:underline">${label}</a>`;
    })
    .replace(/\n/g, "<br>");
}

function el(tag: string, cls: string): HTMLElement {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  return e;
}
