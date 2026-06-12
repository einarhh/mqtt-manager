// Decode a base64 string (as sent by the Go backend) into raw bytes.
export function b64ToBytes(b64: string): Uint8Array {
  try {
    const bin = atob(b64);
    return Uint8Array.from(bin, (c) => c.charCodeAt(0));
  } catch {
    return new Uint8Array(0);
  }
}

// Decode a base64 string (as sent by the Go backend) into UTF-8 text.
export function b64ToText(b64: string): string {
  try {
    return new TextDecoder("utf-8", { fatal: false }).decode(b64ToBytes(b64));
  } catch {
    return "";
  }
}

// Lowercase hex string of a byte array, e.g. "0a1b2c".
export function bytesToHex(bytes: Uint8Array): string {
  let s = "";
  for (const b of bytes) s += b.toString(16).padStart(2, "0");
  return s;
}

// topicMatch reports whether an MQTT topic matches a subscription-style filter
// using the `+` (single level) and `#` (multi level, trailing) wildcards.
export function topicMatch(filter: string, topic: string): boolean {
  if (filter === topic) return true;
  const f = filter.split("/");
  const t = topic.split("/");
  for (let i = 0; i < f.length; i++) {
    const seg = f[i];
    if (seg === "#") return true; // matches this level and everything below
    if (i >= t.length) return false;
    if (seg === "+") continue;
    if (seg !== t[i]) return false;
  }
  return f.length === t.length;
}

// Encode UTF-8 text into base64 for transport to the Go backend.
export function textToB64(s: string): string {
  const bytes = new TextEncoder().encode(s);
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin);
}

// Attempt to pretty-print a JSON payload; falls back to the raw text.
export function prettyJSON(text: string): { pretty: string; isJSON: boolean } {
  const trimmed = text.trim();
  if (!trimmed || (trimmed[0] !== "{" && trimmed[0] !== "[")) {
    return { pretty: text, isJSON: false };
  }
  try {
    return { pretty: JSON.stringify(JSON.parse(trimmed), null, 2), isJSON: true };
  } catch {
    return { pretty: text, isJSON: false };
  }
}

// Format a unix-millis timestamp as HH:MM:SS.mmm in local time.
export function formatTime(ts: number): string {
  const d = new Date(ts);
  const t = d.toLocaleTimeString(undefined, { hour12: false });
  return `${t}.${String(d.getMilliseconds()).padStart(3, "0")}`;
}

// Collapse a payload to a short single-line preview for the tree.
export function preview(text: string, max = 64): string {
  const oneLine = text.replace(/\s+/g, " ").trim();
  return oneLine.length > max ? oneLine.slice(0, max) + "…" : oneLine;
}
