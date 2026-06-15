import { writable } from "svelte/store";
import { b64ToText } from "./util";

export interface HistoryEntry {
  ts: number;
  text: string;
  raw: string; // base64 payload, kept so decoder plugins can read raw bytes
  qos: number;
  retained: boolean;
}

// A node in the topic tree. A node can be both a branch (has children) and a
// leaf with its own last value (matching MQTT's hierarchical topics).
export interface TreeNode {
  name: string; // this segment, e.g. "temp"
  path: string; // full topic up to here, e.g. "home/room/temp"
  children: Map<string, TreeNode>;
  text: string | null; // last decoded payload, or null if no value yet
  raw: string | null; // last raw payload (base64), or null if no value yet
  qos: number;
  retained: boolean;
  ts: number | null; // last update (unix millis)
  count: number; // messages received at this exact topic
  history: HistoryEntry[]; // bounded ring buffer, newest last
}

export interface IncomingMessage {
  topic: string;
  payload: string; // base64
  qos: number;
  retained: boolean;
  ts: number;
}

export interface ConnStatus {
  status: string;
  detail: string;
}

// How sibling topics are ordered in the tree. "received" keeps the order they
// first arrived (the Map's insertion order); "alpha" sorts by name.
export type SortMode = "received" | "alpha";

const SORT_KEY = "mqtt-manager:sort";

function readStoredSort(): SortMode {
  return localStorage.getItem(SORT_KEY) === "alpha" ? "alpha" : "received";
}

export const sortMode = writable<SortMode>(readStoredSort());
sortMode.subscribe((m) => localStorage.setItem(SORT_KEY, m));

// Order a list of sibling nodes according to the active sort mode. "received"
// preserves the source order (callers pass nodes straight from the Map).
export function sortNodes(nodes: TreeNode[], mode: SortMode): TreeNode[] {
  return mode === "alpha"
    ? nodes.sort((a, b) => a.name.localeCompare(b.name))
    : nodes;
}

// Kept deep enough to give the numeric chart a usable window of points.
const HISTORY_LIMIT = 500;

function makeNode(name: string, path: string): TreeNode {
  return {
    name,
    path,
    children: new Map(),
    text: null,
    raw: null,
    qos: 0,
    retained: false,
    ts: null,
    count: 0,
    history: [],
  };
}

// A Conn holds all received state for one connection. Its `root` tree is mutated
// in place; the active connection's tree is mirrored into the `tree` store below
// so the tree/detail/publish components can stay connection-agnostic.
export interface Conn {
  id: string;
  name: string;
  root: TreeNode;
  selectedPath: string | null;
  topicCount: number;
  msgCount: number;
  status: ConnStatus;
}

// The connection registry. `conns` is the source of truth; `connections` mirrors
// it as an array for the list UI and is bumped on add/remove/name/status changes.
const conns = new Map<string, Conn>();
export const connections = writable<Conn[]>([]);
export const activeId = writable<string | null>(null);

// View mirrors of the active connection. Existing components subscribe to these
// without needing to know which connection is active.
export const tree = writable<TreeNode>(makeNode("", ""));
export const selectedPath = writable<string | null>(null);
export const totalMessages = writable(0);
export const totalTopics = writable(0);
export const status = writable<ConnStatus>({ status: "disconnected", detail: "" });

let activeIdValue: string | null = null;
activeId.subscribe((v) => (activeIdValue = v));

// Persist the active connection's selection so each remembers its own.
selectedPath.subscribe((v) => {
  if (activeIdValue) {
    const c = conns.get(activeIdValue);
    if (c) c.selectedPath = v;
  }
});

function bumpConnections(): void {
  connections.set([...conns.values()]);
}

// addConnection registers a connection (or updates its name if it already
// exists). The first connection added becomes the active view.
export function addConnection(id: string, name: string): void {
  let c = conns.get(id);
  if (!c) {
    c = {
      id,
      name,
      root: makeNode("", ""),
      selectedPath: null,
      topicCount: 0,
      msgCount: 0,
      status: { status: "connecting", detail: "" },
    };
    conns.set(id, c);
  } else {
    c.name = name;
  }
  bumpConnections();
  if (activeIdValue === null) setActive(id);
}

// removeConnection forgets a connection's captured data. If it was active, the
// view falls back to another connection (or empty).
export function removeConnection(id: string): void {
  if (!conns.delete(id)) return;
  bumpConnections();
  if (activeIdValue === id) {
    const next = conns.keys().next();
    setActive(next.done ? null : next.value);
  }
}

// setActive switches which connection's data is shown in the view mirrors.
export function setActive(id: string | null): void {
  activeId.set(id);
  if (!id) {
    tree.set(makeNode("", ""));
    totalMessages.set(0);
    totalTopics.set(0);
    status.set({ status: "disconnected", detail: "" });
    selectedPath.set(null);
    return;
  }
  const c = conns.get(id);
  if (!c) return;
  tree.set(c.root);
  totalMessages.set(c.msgCount);
  totalTopics.set(c.topicCount);
  status.set(c.status);
  selectedPath.set(c.selectedPath);
}

// setStatus records a connection's status and reflects it in the view if active.
export function setStatus(id: string, s: ConnStatus): void {
  const c = conns.get(id);
  if (!c) return;
  c.status = s;
  bumpConnections();
  if (id === activeIdValue) status.set(s);
}

// ingest folds a batch of incoming messages into the given connection's tree.
export function ingest(id: string, batch: IncomingMessage[]): void {
  const c = conns.get(id);
  if (!c) return; // untracked (e.g. removed) — drop
  for (const m of batch) {
    let node = c.root;
    let path = "";
    for (const seg of m.topic.split("/")) {
      path = path ? `${path}/${seg}` : seg;
      let child = node.children.get(seg);
      if (!child) {
        child = makeNode(seg, path);
        node.children.set(seg, child);
      }
      node = child;
    }
    const wasEmpty = node.ts === null;
    const text = b64ToText(m.payload);
    node.text = text;
    node.raw = m.payload;
    node.qos = m.qos;
    node.retained = m.retained;
    node.ts = m.ts;
    node.count++;
    node.history.push({ ts: m.ts, text, raw: m.payload, qos: m.qos, retained: m.retained });
    if (node.history.length > HISTORY_LIMIT) node.history.shift();
    if (wasEmpty) c.topicCount++;
    c.msgCount++;
  }
  if (id === activeIdValue) {
    totalMessages.set(c.msgCount);
    totalTopics.set(c.topicCount);
    tree.set(c.root);
  }
}

// clearConnection wipes a connection's received data, keeping the connection
// itself (e.g. before a reconnect reuses its client).
export function clearConnection(id: string): void {
  const c = conns.get(id);
  if (!c) return;
  c.root.children.clear();
  c.topicCount = 0;
  c.msgCount = 0;
  c.selectedPath = null;
  if (id === activeIdValue) {
    totalMessages.set(0);
    totalTopics.set(0);
    selectedPath.set(null);
    tree.set(c.root);
  }
}

// clearTree wipes the active connection's received data (the toolbar action).
export function clearTree(): void {
  if (activeIdValue) clearConnection(activeIdValue);
}

// findNode resolves a full topic path to its node in the active connection's
// tree, or null.
export function findNode(path: string | null): TreeNode | null {
  if (!path || !activeIdValue) return null;
  const c = conns.get(activeIdValue);
  if (!c) return null;
  let node: TreeNode | undefined = c.root;
  for (const seg of path.split("/")) {
    node = node.children.get(seg);
    if (!node) return null;
  }
  return node;
}
