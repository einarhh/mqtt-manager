import { writable } from "svelte/store";
import { b64ToText } from "./util";

export interface HistoryEntry {
  ts: number;
  text: string;
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

const HISTORY_LIMIT = 100;

function makeNode(name: string, path: string): TreeNode {
  return {
    name,
    path,
    children: new Map(),
    text: null,
    qos: 0,
    retained: false,
    ts: null,
    count: 0,
    history: [],
  };
}

// The authoritative tree lives outside Svelte and is mutated in place; the
// `tree` store is bumped after each batch to trigger re-render.
export const root = makeNode("", "");
export const tree = writable<TreeNode>(root);
export const selectedPath = writable<string | null>(null);
export const totalMessages = writable(0);
export const totalTopics = writable(0);
export const status = writable<ConnStatus>({ status: "disconnected", detail: "" });

let topicCount = 0;
let msgCount = 0;

// ingest folds a batch of incoming messages into the tree.
export function ingest(batch: IncomingMessage[]): void {
  for (const m of batch) {
    let node = root;
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
    node.qos = m.qos;
    node.retained = m.retained;
    node.ts = m.ts;
    node.count++;
    node.history.push({ ts: m.ts, text, qos: m.qos, retained: m.retained });
    if (node.history.length > HISTORY_LIMIT) node.history.shift();
    if (wasEmpty) topicCount++;
    msgCount++;
  }
  totalMessages.set(msgCount);
  totalTopics.set(topicCount);
  tree.set(root);
}

// clearTree wipes all received data (e.g. on disconnect / new connection).
export function clearTree(): void {
  root.children.clear();
  topicCount = 0;
  msgCount = 0;
  totalMessages.set(0);
  totalTopics.set(0);
  selectedPath.set(null);
  tree.set(root);
}

// findNode resolves a full topic path to its node, or null.
export function findNode(path: string | null): TreeNode | null {
  if (!path) return null;
  let node: TreeNode | undefined = root;
  for (const seg of path.split("/")) {
    node = node.children.get(seg);
    if (!node) return null;
  }
  return node;
}
