<script lang="ts">
  import { status, selectedPath } from "./stores";
  import { textToB64 } from "./util";
  import { Publish } from "../../wailsjs/go/main/App";

  let topic = "";
  let payload = "";
  let qos = 0;
  let retain = false;
  let error = "";
  let flash = false;

  $: connected = $status.status === "connected";

  // Convenience: drop the currently selected topic into the publish box.
  function useSelected() {
    if ($selectedPath) topic = $selectedPath;
  }

  async function publish() {
    error = "";
    if (!topic.trim()) {
      error = "Topic is required";
      return;
    }
    try {
      await Publish(topic, textToB64(payload), Number(qos), retain);
      flash = true;
      setTimeout(() => (flash = false), 500);
    } catch (e) {
      error = String(e);
    }
  }
</script>

<div class="publish">
  <div class="header">
    <span class="title">Publish</span>
    <button class="link" on:click={useSelected} disabled={!$selectedPath}>
      Use selected
    </button>
  </div>

  <input class="topic" type="text" placeholder="topic/to/publish" bind:value={topic} />
  <textarea
    class="payload"
    placeholder="payload (text or JSON)"
    rows="4"
    bind:value={payload}
  ></textarea>

  <div class="controls">
    <label class="qos">
      QoS
      <select bind:value={qos}>
        <option value={0}>0</option>
        <option value={1}>1</option>
        <option value={2}>2</option>
      </select>
    </label>
    <label class="retain">
      <input type="checkbox" bind:checked={retain} />
      Retain
    </label>
    <button
      class="primary"
      class:flash
      on:click={publish}
      disabled={!connected}
      title={connected ? "Publish to broker" : "Connect to a broker first"}
    >
      Publish
    </button>
  </div>

  {#if error}<div class="error">{error}</div>{/if}
</div>

<style>
  .publish {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    border-top: 1px solid var(--border);
  }
  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .title {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-dim);
  }
  .topic {
    font-family: var(--mono);
  }
  .payload {
    font-family: var(--mono);
    resize: vertical;
  }
  .controls {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .qos,
  .retain {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: var(--text-dim);
  }
  .primary {
    margin-left: auto;
  }
  .primary.flash {
    background: var(--ok);
  }
  .link {
    background: none;
    border: none;
    color: var(--accent);
    cursor: pointer;
    font-size: 12px;
    padding: 0;
  }
  .link:disabled {
    color: var(--text-dim);
    cursor: default;
  }
  .error {
    color: var(--err);
    font-size: 12px;
  }
</style>
