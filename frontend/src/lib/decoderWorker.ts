// Web Worker that compiles and runs decoder plugins off the UI thread.
//
// Plugins are authored as a single `export default <object>` module (see the
// README). We don't use dynamic `import()` of a blob — WKWebView support for
// that is patchy — so instead the leading `export default` is rewritten to a
// `return` and the module body is evaluated with `new Function`. Consequence:
// a plugin file is plain JavaScript with helper declarations first and exactly
// one trailing `export default`; no other import/export statements.
//
// Running here means a buggy or slow plugin can be watchdog-terminated by the
// main thread (an infinite loop can't be interrupted any other way) without
// freezing the app.

interface Ctx {
  topic: string;
  bytes: Uint8Array;
  hex: string;
  text: string;
  ts: number | null;
}

interface CompiledPlugin {
  id: string;
  name: string;
  topic: string;
  hasMatch: boolean;
  mod: {
    name?: string;
    topic?: string;
    match?: (ctx: Ctx) => unknown;
    decode: (ctx: Ctx) => unknown;
  };
}

const compiled = new Map<string, CompiledPlugin>();

function errMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  return String(err);
}

function compile(source: string): CompiledPlugin["mod"] {
  // Rewrite the single `export default ...` into a return statement.
  const transformed = source.replace(/export\s+default\s+/, "return ");
  // eslint-disable-next-line no-new-func
  const factory = new Function(`"use strict";\n${transformed}\n`);
  const mod = factory();
  if (!mod || typeof mod !== "object") {
    throw new Error("plugin default export must be an object");
  }
  if (typeof mod.decode !== "function") {
    throw new Error("plugin must export a decode(ctx) function");
  }
  return mod;
}

function hexToBytes(hex: string): Uint8Array {
  const len = hex.length >> 1;
  const out = new Uint8Array(len);
  for (let i = 0; i < len; i++) {
    out[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

// Guarantee the result is structured-cloneable and free of functions by JSON
// round-tripping it. Decoder results are plain data, so nothing useful is lost.
function safeResult(result: unknown): unknown {
  try {
    return JSON.parse(JSON.stringify(result ?? null));
  } catch (err) {
    return { error: "decode() returned a non-serialisable value: " + errMessage(err) };
  }
}

self.onmessage = (e: MessageEvent) => {
  const msg = e.data;

  if (msg.type === "load") {
    compiled.clear();
    const metas = [];
    for (const p of msg.plugins as Array<{ id: string; name: string; source: string }>) {
      try {
        const mod = compile(p.source);
        const topic = typeof mod.topic === "string" && mod.topic ? mod.topic : "#";
        compiled.set(p.id, {
          id: p.id,
          name: mod.name || p.name,
          topic,
          hasMatch: typeof mod.match === "function",
          mod,
        });
        metas.push({ id: p.id, ok: true, name: mod.name || p.name, topic });
      } catch (err) {
        metas.push({ id: p.id, ok: false, error: errMessage(err) });
      }
    }
    (self as unknown as Worker).postMessage({ type: "loaded", reqId: msg.reqId, metas });
    return;
  }

  if (msg.type === "decode") {
    const c = compiled.get(msg.pluginId);
    if (!c) {
      (self as unknown as Worker).postMessage({ type: "decoded", reqId: msg.reqId, status: "nomatch" });
      return;
    }
    const bytes = hexToBytes(msg.hex);
    const ctx: Ctx = {
      topic: msg.topic,
      bytes,
      hex: msg.hex,
      text: new TextDecoder("utf-8", { fatal: false }).decode(bytes),
      ts: msg.ts ?? null,
    };
    try {
      if (c.hasMatch && !c.mod.match!(ctx)) {
        (self as unknown as Worker).postMessage({ type: "decoded", reqId: msg.reqId, status: "nomatch" });
        return;
      }
      const result = c.mod.decode(ctx);
      (self as unknown as Worker).postMessage({
        type: "decoded",
        reqId: msg.reqId,
        status: "ok",
        result: safeResult(result),
      });
    } catch (err) {
      (self as unknown as Worker).postMessage({
        type: "decoded",
        reqId: msg.reqId,
        status: "error",
        error: errMessage(err),
      });
    }
  }
};

export {};
