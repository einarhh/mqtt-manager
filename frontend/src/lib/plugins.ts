// Decoder-plugin runtime and manager state.
//
// Plugins are stored on disk by the Go backend (see internal/plugins) and run
// in a Web Worker (see decoderWorker.ts). This module is the bridge: it loads
// the records, compiles the enabled ones in the worker, and exposes a
// `decodeNode` used by the detail panel plus save/delete/enable helpers used by
// the plugin manager UI.

import { writable, get } from "svelte/store";
import {
  ListPlugins,
  SavePlugin as goSavePlugin,
  DeletePlugin as goDeletePlugin,
  ImportPlugin as goImportPlugin,
  ExportPlugin as goExportPlugin,
} from "../../wailsjs/go/main/App";
import { topicMatch, b64ToBytes, bytesToHex } from "./util";
import type { TreeNode } from "./stores";
import DecoderWorker from "./decoderWorker?worker";

export interface DecodeField {
  label: string;
  value: unknown;
  hint?: string;
}

// What a plugin's decode() returns, annotated with which plugin produced it.
// `html` is an escape hatch: when present it is rendered verbatim (themed via the
// app's CSS variables) and takes precedence over the structured fields/json/text.
export interface DecodeResult {
  pluginId: string;
  pluginName: string;
  summary?: string;
  fields?: DecodeField[];
  json?: unknown;
  text?: string;
  html?: string;
  error?: string;
}

// A plugin record plus runtime status (parsed topic / compile error).
export interface PluginInfo {
  id: string;
  name: string;
  filename: string;
  enabled: boolean;
  order: number;
  source: string;
  topic: string; // match filter parsed from the compiled module ("#" if unknown)
  scope: string; // "topic" (per-message) or "subtree" (aggregate over descendants)
  loadError: string; // compile/load error, empty when healthy
}

export const pluginList = writable<PluginInfo[]>([]);

const DECODE_TIMEOUT_MS = 800;
const LOAD_TIMEOUT_MS = 3000;

let worker: Worker | null = null;
let nextReq = 1;
const pending = new Map<number, (msg: any) => void>();

// Decode results cached by `path:count` so re-renders and re-selection are free.
const cache = new Map<string, DecodeResult | null>();

function ensureWorker(): Worker {
  if (worker) return worker;
  const w = new DecoderWorker();
  w.onmessage = (e: MessageEvent) => {
    const msg = e.data;
    const resolve = pending.get(msg.reqId);
    if (resolve) {
      pending.delete(msg.reqId);
      resolve(msg);
    }
  };
  worker = w;
  return w;
}

// Terminate the worker (e.g. after a hung plugin) and fail any in-flight calls.
function resetWorker(): void {
  if (worker) {
    worker.terminate();
    worker = null;
  }
  for (const [, resolve] of pending) resolve({ timedOut: true });
  pending.clear();
}

function send(msg: any, timeoutMs: number): Promise<any> {
  const w = ensureWorker();
  const reqId = nextReq++;
  return new Promise((resolve) => {
    const timer = setTimeout(() => {
      if (pending.delete(reqId)) resolve({ timedOut: true });
    }, timeoutMs);
    pending.set(reqId, (m) => {
      clearTimeout(timer);
      resolve(m);
    });
    w.postMessage({ ...msg, reqId });
  });
}

// Compile the currently-enabled, healthy plugins into the (fresh) worker.
async function loadIntoWorker(records: PluginInfo[]): Promise<void> {
  const enabled = records.filter((r) => r.enabled && !r.loadError);
  if (!enabled.length) return;
  const resp = await send(
    {
      type: "load",
      plugins: enabled.map((r) => ({ id: r.id, name: r.name, source: r.source })),
    },
    LOAD_TIMEOUT_MS,
  );
  if (resp?.metas) {
    const byId = new Map<string, any>(resp.metas.map((m: any) => [m.id, m]));
    for (const r of records) {
      const m = byId.get(r.id);
      if (!m) continue;
      if (m.ok) {
        r.topic = m.topic || "#";
        r.scope = m.scope === "subtree" ? "subtree" : "topic";
        r.loadError = "";
      } else {
        r.loadError = m.error || "failed to load";
        r.enabled = false;
      }
    }
  } else if (resp?.timedOut) {
    for (const r of enabled) r.loadError = "loading timed out";
  }
}

// reload pulls the plugin records from disk and recompiles them in a fresh
// worker. Call after any change (save/delete/enable) and once on startup.
export async function reload(): Promise<void> {
  cache.clear();
  resetWorker();

  let records: PluginInfo[] = [];
  try {
    const list = await ListPlugins();
    records = list.map((p) => ({
      id: p.id,
      name: p.name,
      filename: p.filename,
      enabled: p.enabled,
      order: p.order,
      source: p.source,
      topic: "#",
      scope: "topic",
      loadError: "",
    }));
  } catch {
    records = [];
  }

  await loadIntoWorker(records);
  pluginList.set(records);
}

function disablePlugin(id: string, reason: string): void {
  pluginList.update((list) =>
    list.map((p) => (p.id === id ? { ...p, enabled: false, loadError: reason } : p)),
  );
}

// runCandidates sends `payload` to each candidate plugin in turn and returns the
// first that claims it (or null). Shared by per-message and subtree decoding.
async function runCandidates(
  candidates: PluginInfo[],
  payload: Record<string, unknown>,
  cacheKey: string,
): Promise<DecodeResult | null> {
  if (cache.has(cacheKey)) return cache.get(cacheKey) ?? null;
  if (!candidates.length) {
    cache.set(cacheKey, null);
    return null;
  }

  let result: DecodeResult | null = null;
  for (const c of candidates) {
    const resp = await send({ type: "decode", pluginId: c.id, ...payload }, DECODE_TIMEOUT_MS);

    if (resp?.timedOut) {
      // The plugin hung; disable it for this session and rebuild the worker.
      disablePlugin(c.id, "decode timed out — disabled for this session");
      resetWorker();
      await loadIntoWorker(get(pluginList));
      continue;
    }
    if (!resp || resp.status === "nomatch") continue;
    if (resp.status === "error") {
      result = { pluginId: c.id, pluginName: c.name, error: resp.error || "decode error" };
      break;
    }
    const r = resp.result || {};
    result = {
      pluginId: c.id,
      pluginName: c.name,
      summary: r.summary,
      fields: Array.isArray(r.fields) ? r.fields : undefined,
      json: r.json,
      text: typeof r.text === "string" ? r.text : undefined,
      html: typeof r.html === "string" ? r.html : undefined,
      error: typeof r.error === "string" ? r.error : undefined,
    };
    break;
  }

  cache.set(cacheKey, result);
  return result;
}

// decodeRaw runs the first enabled per-message plugin whose topic filter (and
// optional match() predicate) claims a base64 payload on `topic`. `cacheKey`
// should be unique per (payload, plugin set). Returns null when no plugin
// applies, or a DecodeResult (which may carry an `error`).
export async function decodeRaw(
  topic: string,
  rawB64: string | null,
  cacheKey: string,
  ts: number | null = null,
): Promise<DecodeResult | null> {
  if (rawB64 === null) return null;
  if (cache.has(cacheKey)) return cache.get(cacheKey) ?? null;
  const candidates = get(pluginList).filter(
    (p) => p.enabled && !p.loadError && p.scope !== "subtree" && topicMatch(p.topic, topic),
  );
  const hex = candidates.length ? bytesToHex(b64ToBytes(rawB64)) : "";
  return runCandidates(candidates, { topic, hex, ts }, cacheKey);
}

// decodeNode decodes a tree node's latest payload.
export function decodeNode(node: TreeNode): Promise<DecodeResult | null> {
  if (!node || node.raw === null) return Promise.resolve(null);
  return decodeRaw(node.path, node.raw, `${node.path}:${node.count}`, node.ts);
}

// decodeSubtree runs the first enabled subtree plugin whose topic filter claims
// `topic`, giving it a snapshot of descendant topics' latest values. `children`
// maps a relative topic path to its latest { text, raw (base64), ts }.
export async function decodeSubtree(
  topic: string,
  children: Record<string, { text: string | null; raw: string | null; ts: number | null }>,
  cacheKey: string,
): Promise<DecodeResult | null> {
  if (cache.has(cacheKey)) return cache.get(cacheKey) ?? null;
  const candidates = get(pluginList).filter(
    (p) => p.enabled && !p.loadError && p.scope === "subtree" && topicMatch(p.topic, topic),
  );
  if (!candidates.length) {
    cache.set(cacheKey, null);
    return null;
  }
  // Convert each child's base64 payload to hex for the worker.
  const payload: Record<string, { text: string | null; hex: string; ts: number | null }> = {};
  for (const k in children) {
    const v = children[k];
    payload[k] = { text: v.text, hex: v.raw ? bytesToHex(b64ToBytes(v.raw)) : "", ts: v.ts };
  }
  return runCandidates(candidates, { topic, children: payload }, cacheKey);
}

// --- manager actions ------------------------------------------------------

// toGoPlugin strips runtime-only fields before persisting.
function toGoPlugin(p: Partial<PluginInfo>): any {
  return {
    id: p.id ?? "",
    name: p.name ?? "",
    filename: p.filename ?? "",
    enabled: p.enabled ?? false,
    order: p.order ?? 0,
    source: p.source ?? "",
  };
}

export async function savePlugin(p: Partial<PluginInfo>): Promise<PluginInfo> {
  const saved = (await goSavePlugin(toGoPlugin(p))) as PluginInfo;
  await reload();
  return saved;
}

export async function deletePlugin(id: string): Promise<void> {
  await goDeletePlugin(id);
  await reload();
}

export async function setPluginEnabled(p: PluginInfo, enabled: boolean): Promise<void> {
  await goSavePlugin(toGoPlugin({ ...p, enabled }));
  await reload();
}

// importPlugin prompts for a .js file and adds it as a new plugin. Returns the
// imported plugin, or null if the user cancelled the dialog.
export async function importPlugin(): Promise<PluginInfo | null> {
  const p = (await goImportPlugin()) as PluginInfo;
  if (!p || !p.id) return null; // cancelled
  await reload();
  return p;
}

// exportPlugin prompts for a destination and writes the plugin's .js file.
export async function exportPlugin(id: string): Promise<void> {
  await goExportPlugin(id);
}
