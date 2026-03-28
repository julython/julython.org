/**
 * Dedicated WebLLM worker handler.
 * This file is bundled separately and spawned as a Web Worker by llm.ts.
 * It does nothing except wire the WebWorkerMLCEngineHandler to self.onmessage.
 */
import { WebWorkerMLCEngineHandler } from "@mlc-ai/web-llm";

const handler = new WebWorkerMLCEngineHandler();
self.onmessage = (msg: MessageEvent) => {
  handler.onmessage(msg);
};