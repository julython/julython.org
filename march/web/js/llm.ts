/**
 * LLM abstraction: tries window.ai (Chrome Gemini Nano) first,
 * falls back to WebLLM (Qwen2.5-Coder-1.5B via WebGPU).
 *
 * Both backends expose the same interface: stream tokens via onChunk,
 * resolve with the full response string.
 */

// ── Types ─────────────────────────────────────────────────────────────────────

export interface LLMSession {
  /** onChunk receives the full accumulated text so far, not just the new token. */
  prompt(text: string, onChunk: (fullText: string) => void): Promise<string>;
  destroy(): void;
}

// ── window.ai (Chrome 127+, Gemini Nano) ─────────────────────────────────────

// Gemini Nano has a small context window — keep system prompt tight.
const MAX_SYSTEM_PROMPT_CHARS = 2000;

async function tryWindowAI(systemPrompt: string): Promise<LLMSession | null> {
  const w = window as any;
  const api = w.LanguageModel ?? w.ai?.languageModel;
  if (!api) return null;

  const avail: string = await api.availability().catch(() => "unavailable");
  console.debug("[window.ai] availability:", avail);
  if (avail === "unavailable" || avail === "no") return null;

  const truncated = systemPrompt.slice(0, MAX_SYSTEM_PROMPT_CHARS);
  const session = await api.create({
    systemPrompt: truncated,
    // initialPrompts is more reliably respected than systemPrompt alone —
    // seed the conversation with context as an established assistant turn.
    initialPrompts: [
      { role: "user", content: "What is the context for our conversation?" },
      { role: "assistant", content: truncated },
    ],
  }).catch((e: unknown) => {
    console.warn("[window.ai] create failed:", e);
    return null;
  });
  if (!session) return null;

  // Also keep the full context to inject into each prompt in case
  // the system prompt was truncated.
  const context = systemPrompt;

  return {
    async prompt(text, onChunk) {
      // Prepend a short context reminder so Gemini Nano can't forget it.
      const fullPrompt = `Regarding the repo "${context.slice(0, 100).split('\n')[0]}":\n${text}`;

      const stream = session.promptStreaming(fullPrompt);
      let full = "";
      for await (const chunk of stream) {
        if (chunk.startsWith(full)) {
          full = chunk;
        } else {
          full += chunk;
        }
        onChunk(full);
      }
      return full;
    },
    destroy() { session.destroy(); },
  };
}

// ── WebLLM fallback (Qwen2.5-Coder-1.5B, runs via WebGPU) ───────────────────
// Loaded from CDN dynamically so it doesn't bloat the main bundle.
// ~1GB model download on first use, cached in browser cache thereafter.

const WEBLLM_CDN = "https://esm.run/@mlc-ai/web-llm";
const WEBLLM_MODEL = "Qwen2.5-Coder-1.5B-Instruct-q4f16_1-MLC";

let webllmEngine: any = null; // reuse across sessions

async function tryWebLLM(
  systemPrompt: string,
  onProgress?: (msg: string) => void,
): Promise<LLMSession | null> {
  try {
    const { CreateMLCEngine } = await import(/* @vite-ignore */ WEBLLM_CDN);

    if (!webllmEngine) {
      onProgress?.("Loading Qwen2.5-Coder model (first load may take a minute)…");
      webllmEngine = await CreateMLCEngine(WEBLLM_MODEL, {
        initProgressCallback: (p: any) => {
          onProgress?.(`Loading model: ${Math.round((p.progress ?? 0) * 100)}%`);
        },
      });
    }

    return {
      async prompt(text, onChunk) {
        const messages = [
          { role: "system", content: systemPrompt },
          { role: "user", content: text },
        ];
          let full = "";
        const stream = await webllmEngine.chat.completions.create({ messages, stream: true });
        for await (const chunk of stream) {
          const token = chunk.choices[0]?.delta?.content ?? "";
          if (token) {
            full += token;
            onChunk(full);
          }
        }
        return full;
      },
      destroy() { /* keep engine alive for reuse */ },
    };
  } catch (e) {
    console.warn("WebLLM unavailable:", e);
    return null;
  }
}

// ── Public API ────────────────────────────────────────────────────────────────

/**
 * Create an LLM session. Tries window.ai first, then WebLLM.
 * onProgress is called with status messages during model loading.
 */
export async function createSession(
  systemPrompt: string,
  onProgress?: (msg: string) => void,
): Promise<LLMSession> {
  onProgress?.("Checking for browser AI…");

  const windowAI = await tryWindowAI(systemPrompt);
  if (windowAI) {
    onProgress?.("Using Chrome built-in AI (Gemini Nano)");
    return windowAI;
  }

  onProgress?.("Chrome AI unavailable, loading WebLLM…");
  const webllm = await tryWebLLM(systemPrompt, onProgress);
  if (webllm) return webllm;

  throw new Error(
    "No LLM available. Enable Chrome AI at chrome://flags/#optimization-guide-on-device-model, " +
    "or use a WebGPU-capable browser."
  );
}

/**
 * Build the system prompt from scorecard + file contents.
 */
export function buildSystemPrompt(
  repoName: string,
  score: number,
  categories: Array<{ name: string; score: number; max: number; signals: string[] }>,
  files: Record<string, string>,
): string {
  const scorecard = categories
    .map(c => `- ${c.name}: ${c.score}/${c.max} (${c.signals.join(", ") || "no signals detected"})`)
    .join("\n");

  const fileDump = Object.entries(files)
    .map(([path, content]) => `### ${path}\n\`\`\`\n${content.slice(0, 500)}\n\`\`\``)
    .slice(0, 5)  // at most 5 files
    .join("\n\n");

  return `You are a code quality assistant analyzing the GitHub repository "${repoName}".
Answer questions about this repo concisely. Focus on actionable advice.
If asked about something not covered by the files below, say so honestly.

## Scorecard (${score}/100)
${scorecard}

## Key Files
${fileDump || "No files were fetched."}`;
}