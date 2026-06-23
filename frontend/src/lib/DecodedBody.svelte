<script lang="ts">
  // The single place plugin output is rendered. A plugin either returns its own
  // `html` (rendered verbatim, themed via the app's CSS variables) or the
  // structured fields/json/text, which this component lays out consistently.
  import type { DecodeResult } from "./plugins";

  export let result: DecodeResult;

  function fmtValue(v: unknown): string {
    if (v === null || v === undefined) return "";
    if (typeof v === "object") return JSON.stringify(v);
    return String(v);
  }
</script>

{#if result.html}
  <!-- Escape hatch: the plugin owns this region. Author-trusted markup (same
       trust model as the JS decoders); scripts do not execute. -->
  <div class="plugin-html">{@html result.html}</div>
{:else}
  {#if result.fields && result.fields.length}
    <table class="fields">
      <tbody>
        {#each result.fields as f}
          <tr>
            <td class="flabel">{f.label}</td>
            <td class="fvalue">
              {fmtValue(f.value)}{#if f.hint}<span class="fhint"> {f.hint}</span>{/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
  {#if result.json !== undefined && result.json !== null}
    <pre class="value sub">{JSON.stringify(result.json, null, 2)}</pre>
  {/if}
  {#if result.text}<pre class="value sub">{result.text}</pre>{/if}
{/if}

<style>
  .fields {
    width: 100%;
    border-collapse: collapse;
    font-family: var(--mono);
    font-size: 12px;
  }
  .fields td {
    padding: 2px 8px 2px 0;
    vertical-align: top;
    border-bottom: 1px solid var(--border);
  }
  .flabel {
    color: var(--text-dim);
    white-space: nowrap;
    width: 1%;
  }
  .fvalue {
    color: var(--text);
    word-break: break-word;
  }
  .fhint {
    color: var(--text-dim);
  }
  .value.sub {
    background: var(--bg-inset);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 10px;
    font-family: var(--mono);
    font-size: 12px;
    color: var(--text);
    white-space: pre-wrap;
    word-break: break-word;
    margin: 8px 0 0;
  }
  /* Sensible defaults for plugin-authored HTML; it can override freely and use
     the app's CSS variables (--accent, --text, --text-dim, --border, --ok,
     --warn, --err, --bg-inset, --bg-hover, --mono) to match the theme. */
  .plugin-html {
    font-size: 13px;
    color: var(--text);
    line-height: 1.5;
  }
  .plugin-html :global(a) {
    color: var(--accent);
  }
  .plugin-html :global(table) {
    border-collapse: collapse;
  }
  .plugin-html :global(code),
  .plugin-html :global(pre) {
    font-family: var(--mono);
  }
</style>
