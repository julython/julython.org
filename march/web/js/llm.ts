import { CreateWebWorkerMLCEngine } from "@mlc-ai/web-llm";
import type { MLCEngineInterface } from "@mlc-ai/web-llm";

// ── Model registry ────────────────────────────────────────────────────────────
// Add new models here as they're released. size is approximate download in MB.

export interface ModelRecord {
  id: string;
  label: string;
  sizeMB: number;
  description: string;
}

export const MODELS: ModelRecord[] = [
  {
    id: "SmolLM2-360M-Instruct-q4f16_1-MLC",
    label: "SmolLM2 360M (fast, ~130MB)",
    sizeMB: 130,
    description: "Fastest option, limited reasoning",
  },
  {
    id: "Llama-3.2-1B-Instruct-q4f16_1-MLC",
    label: "Llama 3.2 1B (~879MB)",
    sizeMB: 879,
    description: "Good balance of speed and quality",
  },
  {
    id: "Llama-3.2-3B-Instruct-q4f16_1-MLC",
    label: "Llama 3.2 3B (~2.2GB)",
    sizeMB: 2263,
    description: "Best quality, requires more VRAM",
  },
];

export const DEFAULT_MODEL = MODELS[1].id; // Llama 1B

// ── Types ─────────────────────────────────────────────────────────────────────

export interface LLMSession {
  /** onChunk receives the full accumulated text so far. */
  prompt(text: string, onChunk: (fullText: string) => void): Promise<string>;
  destroy(): void;
}

// ── Cache helpers ─────────────────────────────────────────────────────────────

/**
 * Returns the approximate total size in MB of cached WebLLM model data,
 * and which model IDs appear to be cached.
 */
export async function getCachedModels(): Promise<{ modelId: string; sizeMB: number }[]> {
  if (!("caches" in window)) return [];
  try {
    const cacheNames = await caches.keys();
    const webllmCaches = cacheNames.filter(n => n.startsWith("webllm/"));
    const results: { modelId: string; sizeMB: number }[] = [];

    for (const name of webllmCaches) {
      const cache = await caches.open(name);
      const keys = await cache.keys();
      let bytes = 0;
      for (const req of keys) {
        const res = await cache.match(req);
        if (res) {
          const blob = await res.blob();
          bytes += blob.size;
        }
      }
      // Cache name is like "webllm/Llama-3.2-1B-Instruct-q4f16_1-MLC"
      const modelId = name.replace("webllm/", "");
      results.push({ modelId, sizeMB: Math.round(bytes / 1024 / 1024) });
    }
    return results;
  } catch {
    return [];
  }
}

/**
 * Delete all WebLLM model caches, or a specific model by ID.
 */
export async function deleteCachedModels(modelId?: string): Promise<void> {
  if (!("caches" in window)) return;
  const cacheNames = await caches.keys();
  const targets = modelId
    ? cacheNames.filter(n => n === `webllm/${modelId}`)
    : cacheNames.filter(n => n.startsWith("webllm/"));
  await Promise.all(targets.map(n => caches.delete(n)));
  // Also reset the engine so next use re-downloads
  webllmEngine = null;
}

// ── window.ai ─────────────────────────────────────────────────────────────────

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
    initialPrompts: [
      { role: "user", content: "What is the context for our conversation?" },
      { role: "assistant", content: truncated },
    ],
  }).catch((e: unknown) => {
    console.warn("[window.ai] create failed:", e);
    return null;
  });
  if (!session) return null;

  const context = systemPrompt;
  return {
    async prompt(text, onChunk) {
      const fullPrompt = `Regarding the repo "${context.slice(0, 100).split("\n")[0]}":\n${text}`;
      const stream = session.promptStreaming(fullPrompt);
      let full = "";
      for await (const chunk of stream) {
        full = chunk.startsWith(full) ? chunk : full + chunk;
        onChunk(full);
      }
      return full;
    },
    destroy() { session.destroy(); },
  };
}

// ── WebLLM ────────────────────────────────────────────────────────────────────

let webllmEngine: MLCEngineInterface | null = null;
let loadedModelId: string | null = null;

async function tryWebLLM(
  systemPrompt: string,
  modelId: string,
  workerURL: string,
  onProgress?: (msg: string) => void,
): Promise<LLMSession | null> {
  try {
    // Reload if model changed
    if (webllmEngine && loadedModelId !== modelId) {
      webllmEngine = null;
      loadedModelId = null;
    }

    if (!webllmEngine) {
      const rec = MODELS.find(m => m.id === modelId);
      onProgress?.(`Loading ${rec?.label ?? modelId}…`);
      webllmEngine = await CreateWebWorkerMLCEngine(
        new Worker(workerURL, { type: "module" }),
        modelId,
        {
          initProgressCallback: (p) => {
            const pct = Math.round((p.progress ?? 0) * 100);
            onProgress?.(`Loading model: ${pct}% — ${p.text ?? ""}`);
          },
        },
      );
      loadedModelId = modelId;
    }

    const sysPrompt = systemPrompt;
    return {
      async prompt(text, onChunk) {
        let full = "";
        const stream = await webllmEngine!.chat.completions.create({
          messages: [
            { role: "system", content: sysPrompt },
            { role: "user", content: text },
          ],
          stream: true,
          temperature: 0.7,
          max_tokens: 512,
        });
        for await (const chunk of stream) {
          const token = chunk.choices[0]?.delta?.content ?? "";
          if (token) { full += token; onChunk(full); }
        }
        return full;
      },
      destroy() { /* keep engine alive across chats */ },
    };
  } catch (e) {
    console.warn("WebLLM unavailable:", e);
    return null;
  }
}

// ── Public API ────────────────────────────────────────────────────────────────

export async function createSession(
  systemPrompt: string,
  modelId: string,
  workerURL: string,
  onProgress?: (msg: string) => void,
): Promise<LLMSession> {
  onProgress?.("Checking for browser AI…");

  const windowAI = await tryWindowAI(systemPrompt);
  if (windowAI) {
    onProgress?.("Using Chrome built-in AI (Gemini Nano)");
    return windowAI;
  }

  onProgress?.("Chrome AI unavailable, loading WebLLM…");
  const webllm = await tryWebLLM(systemPrompt, modelId, workerURL, onProgress);
  if (webllm) return webllm;

  throw new Error(
    "No LLM available. Requires Chrome 127+ with #prompt-api-for-gemini-nano, " +
    "or a WebGPU-capable browser."
  );
}

export function buildSystemPrompt(
  repoName: string,
  score: number,
  categories: Array<{ name: string; score: number; max: number; signals: string[] }>,
  files: Record<string, string>,
): string {
  const scorecard = categories
    .map(c => `- ${c.name}: ${c.score}/${c.max} (${c.signals.join(", ") || "no signals"})`)
    .join("\n");
  const fileDump = Object.entries(files)
    .slice(0, 5)
    .map(([path, content]) => `### ${path}\n\`\`\`\n${content.slice(0, 500)}\n\`\`\``)
    .join("\n\n");
  return `You are a code quality assistant analyzing the GitHub repository "${repoName}".
Answer questions about this repo concisely. Focus on actionable advice.
If asked about something not covered by the files below, say so honestly.

## Scorecard (${score}/100)
${scorecard}

## Key Files
${fileDump || "No files were fetched."}`;
}