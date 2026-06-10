// Decode a base64 string (as sent by the Go backend) into UTF-8 text.
export function b64ToText(b64: string): string {
  try {
    const bin = atob(b64);
    const bytes = Uint8Array.from(bin, (c) => c.charCodeAt(0));
    return new TextDecoder("utf-8", { fatal: false }).decode(bytes);
  } catch {
    return "";
  }
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
