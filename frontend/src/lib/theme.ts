// Light/dark theming. The user picks one of three modes; "system" follows the
// OS setting live. The resolved theme is written to <html data-theme="…">,
// which the token blocks in style.css key off. An inline script in index.html
// applies the same logic before first paint to avoid a flash of wrong theme.
import { writable } from "svelte/store";

export type ThemeMode = "system" | "light" | "dark";
export type ResolvedTheme = "light" | "dark";

const KEY = "mqtt-manager:theme";
const mq = window.matchMedia("(prefers-color-scheme: dark)");

function readStored(): ThemeMode {
  const v = localStorage.getItem(KEY);
  return v === "light" || v === "dark" ? v : "system";
}

function resolve(mode: ThemeMode): ResolvedTheme {
  if (mode === "light" || mode === "dark") return mode;
  return mq.matches ? "dark" : "light";
}

export const themeMode = writable<ThemeMode>(readStored());
// The actual theme in effect (after resolving "system"). Useful for UI labels.
export const resolvedTheme = writable<ResolvedTheme>(resolve(readStored()));

let current: ThemeMode = readStored();

function apply(mode: ThemeMode) {
  const r = resolve(mode);
  document.documentElement.dataset.theme = r;
  resolvedTheme.set(r);
}

themeMode.subscribe((mode) => {
  current = mode;
  if (mode === "system") localStorage.removeItem(KEY);
  else localStorage.setItem(KEY, mode);
  apply(mode);
});

// Re-resolve when the OS theme changes, but only while following it.
mq.addEventListener("change", () => {
  if (current === "system") apply(current);
});

const ORDER: ThemeMode[] = ["system", "light", "dark"];

export function cycleTheme() {
  themeMode.update((m) => ORDER[(ORDER.indexOf(m) + 1) % ORDER.length]);
}
