// Svelte action that replays a CSS animation each time `trigger` changes,
// keeping the element mounted. A `{#key}` block would remount it on every
// message instead, and that DOM churn flickers the surrounding scroll bar.
export function flashOn(
  node: HTMLElement,
  params: { trigger: number; animation: string },
) {
  let prev = params.trigger;
  const play = () => {
    node.style.animation = "none";
    void node.offsetWidth; // force reflow so the animation restarts
    node.style.animation = params.animation;
  };
  // Play once on mount for an already-active node (e.g. a row that just appeared).
  if (params.trigger > 0) play();
  return {
    update(p: { trigger: number; animation: string }) {
      if (p.trigger !== prev) {
        prev = p.trigger;
        play();
      }
    },
  };
}

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

function sameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

// Clock with millisecond precision. A date prefix is added only when the
// timestamp isn't from today (and the year too when it differs), so same-day
// history stays compact while older messages are still unambiguous.
export function formatTime(ts: number): string {
  const d = new Date(ts);
  const t = d.toLocaleTimeString(undefined, { hour12: false });
  const clock = `${t}.${String(d.getMilliseconds()).padStart(3, "0")}`;
  const now = new Date();
  if (sameDay(d, now)) return clock;
  const date = d.toLocaleDateString(
    undefined,
    d.getFullYear() === now.getFullYear()
      ? { month: "short", day: "numeric" }
      : { year: "numeric", month: "short", day: "numeric" },
  );
  return `${date} ${clock}`;
}

export type Coord = { lat: number; lon: number };

const NUM_RE = /^[+-]?\d+(\.\d+)?$/;

function inRange(lat: number, lon: number): Coord | null {
  if (lat < -90 || lat > 90 || lon < -180 || lon > 180) return null;
  return { lat, lon };
}

// Pull the first finite number stored under any of `keys` (case-insensitive).
function pickNum(o: Record<string, unknown>, keys: string[]): number | null {
  for (const k of Object.keys(o)) {
    if (keys.includes(k.toLowerCase())) {
      const n = Number(o[k]);
      if (Number.isFinite(n)) return n;
    }
  }
  return null;
}

// Detect a GPS coordinate in a payload. Accepts a plain "lat,lon" or
// "lat lon" string, or a JSON object with lat/lon-style keys. Returns null
// when the value doesn't look like a coordinate. To avoid matching arbitrary
// number pairs (counters, ratios, …) the plain-string form requires a decimal
// point in at least one component, and both must fall in valid lat/lon ranges.
export function parseCoord(text: string | null | undefined): Coord | null {
  if (!text) return null;
  const trimmed = text.trim();

  // JSON object with recognizable keys, e.g. {"lat":59.9,"lng":10.7}.
  if (trimmed[0] === "{") {
    try {
      const o = JSON.parse(trimmed);
      if (o && typeof o === "object") {
        const lat = pickNum(o, ["lat", "latitude"]);
        const lon = pickNum(o, ["lon", "lng", "long", "longitude"]);
        if (lat !== null && lon !== null) return inRange(lat, lon);
      }
    } catch {
      /* not JSON — fall through */
    }
    return null;
  }

  // Plain "lat,lon" or "lat lon", optionally with a trailing altitude
  // ("lat,lon,alt"). Extra components beyond lat/lon are ignored.
  const parts = trimmed.split(/\s*,\s*|\s+/);
  if (parts.length < 2 || parts.length > 3) return null;
  if (!parts.every((p) => NUM_RE.test(p))) return null;
  if (!parts.slice(0, 2).some((p) => p.includes("."))) return null;
  return inRange(Number(parts[0]), Number(parts[1]));
}

// A browser URL that drops a pin on the given coordinate (OpenStreetMap).
export function mapURL(c: Coord): string {
  return `https://www.openstreetmap.org/?mlat=${c.lat}&mlon=${c.lon}#map=15/${c.lat}/${c.lon}`;
}

// An embeddable OpenStreetMap URL (for an <iframe>) showing a marker at the
// coordinate. `delta` sets the half-span of the viewport in degrees — smaller
// is more zoomed in.
export function mapEmbedURL(c: Coord, delta = 0.01): string {
  const bbox = `${c.lon - delta},${c.lat - delta},${c.lon + delta},${c.lat + delta}`;
  return (
    "https://www.openstreetmap.org/export/embed.html" +
    `?bbox=${encodeURIComponent(bbox)}&layer=mapnik&marker=${c.lat},${c.lon}`
  );
}

// Parse a bare numeric payload into a finite number, or null. Accepts plain
// decimals and scientific notation ("23.4", "-5", "1e3"); rejects anything with
// surrounding non-numeric content so JSON, text, and coordinates don't match.
export function parseNumeric(text: string | null | undefined): number | null {
  if (text === null || text === undefined) return null;
  const trimmed = text.trim();
  if (trimmed === "") return null;
  const n = Number(trimmed);
  return Number.isFinite(n) ? n : null;
}

// Collapse a payload to a short single-line preview for the tree.
export function preview(text: string, max = 64): string {
  const oneLine = text.replace(/\s+/g, " ").trim();
  return oneLine.length > max ? oneLine.slice(0, max) + "…" : oneLine;
}
