<script lang="ts">
  import { formatTime } from "./util";

  // A time-ordered series of numeric samples (oldest first).
  export let series: { ts: number; value: number }[] = [];

  // Drawing surface. Width tracks the container; the viewBox uses these so the
  // SVG scales crisply at any rendered size.
  const W = 600;
  const H = 160;
  const PAD = { top: 10, right: 10, bottom: 18, left: 44 };

  // Plot area in viewBox units.
  $: plotW = W - PAD.left - PAD.right;
  $: plotH = H - PAD.top - PAD.bottom;

  $: values = series.map((s) => s.value);
  $: count = series.length;

  // Value range, padded a little so the line never hugs the top/bottom edge.
  // A flat series gets a synthetic ±1 span so it renders centered.
  $: rawMin = count ? Math.min(...values) : 0;
  $: rawMax = count ? Math.max(...values) : 0;
  $: span = rawMax - rawMin || Math.abs(rawMax) || 1;
  $: yMin = rawMin - span * 0.08;
  $: yMax = rawMax + span * 0.08;

  // Time range. A single point maps to the right edge.
  $: tMin = count ? series[0].ts : 0;
  $: tMax = count ? series[count - 1].ts : 1;
  $: tSpan = tMax - tMin || 1;

  function xOf(ts: number): number {
    return PAD.left + ((ts - tMin) / tSpan) * plotW;
  }
  function yOf(v: number): number {
    return PAD.top + (1 - (v - yMin) / (yMax - yMin)) * plotH;
  }

  // Points as [x, y] pairs in viewBox space, and the SVG path string.
  $: pts = series.map((s) => [xOf(s.ts), yOf(s.value)] as const);
  $: linePath = pts.map(([x, y], i) => `${i ? "L" : "M"}${x.toFixed(1)} ${y.toFixed(1)}`).join(" ");
  // Filled area under the line, closed along the baseline.
  $: areaPath =
    pts.length > 1
      ? `${linePath} L${pts[pts.length - 1][0].toFixed(1)} ${(PAD.top + plotH).toFixed(1)} ` +
        `L${pts[0][0].toFixed(1)} ${(PAD.top + plotH).toFixed(1)} Z`
      : "";

  // A few horizontal gridlines / y-axis ticks.
  $: yTicks = [yMin + (yMax - yMin) * 0, yMin + (yMax - yMin) * 0.5, yMax].map((v) => ({
    v,
    y: yOf(v),
  }));

  function fmtNum(v: number): string {
    // Trim noisy float tails without forcing decimals on integers.
    if (!Number.isFinite(v)) return "—";
    const r = Math.round(v * 1000) / 1000;
    return String(r);
  }

  // Hover state: index of the nearest sample, or null when the pointer is away.
  let hover: number | null = null;
  let svgEl: SVGSVGElement;

  function onMove(e: PointerEvent): void {
    if (!count || !svgEl) return;
    const rect = svgEl.getBoundingClientRect();
    // Map the pointer's pixel x into viewBox units, then to the nearest sample.
    const vbX = ((e.clientX - rect.left) / rect.width) * W;
    let best = 0;
    let bestD = Infinity;
    for (let i = 0; i < pts.length; i++) {
      const d = Math.abs(pts[i][0] - vbX);
      if (d < bestD) {
        bestD = d;
        best = i;
      }
    }
    hover = best;
  }
  function onLeave(): void {
    hover = null;
  }

  $: hoverPt = hover !== null && pts[hover] ? pts[hover] : null;
  $: hoverSample = hover !== null ? series[hover] : null;
  // Keep the tooltip from spilling off the right edge.
  $: tipX = hoverPt ? Math.min(hoverPt[0] + 8, W - 96) : 0;
</script>

<div class="chart">
  <svg
    bind:this={svgEl}
    viewBox="0 0 {W} {H}"
    preserveAspectRatio="none"
    role="img"
    aria-label="Value over time"
    on:pointermove={onMove}
    on:pointerleave={onLeave}
  >
    <!-- gridlines + y labels -->
    {#each yTicks as t}
      <line class="grid" x1={PAD.left} x2={W - PAD.right} y1={t.y} y2={t.y} />
      <text class="axis" x={PAD.left - 6} y={t.y} text-anchor="end" dominant-baseline="middle">
        {fmtNum(t.v)}
      </text>
    {/each}

    {#if count > 1}
      <path class="area" d={areaPath} />
      <path class="line" d={linePath} />
    {/if}

    <!-- sample dots (only when sparse, to avoid clutter) -->
    {#if count <= 60}
      {#each pts as [x, y]}
        <circle class="dot" cx={x} cy={y} r="2" />
      {/each}
    {/if}

    {#if count === 1}
      <circle class="dot solo" cx={pts[0][0]} cy={pts[0][1]} r="3.5" />
    {/if}

    <!-- hover crosshair -->
    {#if hoverPt}
      <line class="cross" x1={hoverPt[0]} x2={hoverPt[0]} y1={PAD.top} y2={PAD.top + plotH} />
      <circle class="cursor" cx={hoverPt[0]} cy={hoverPt[1]} r="3.5" />
    {/if}
  </svg>

  {#if hoverSample}
    <div class="tip" style="left: {(tipX / W) * 100}%">
      <span class="tip-val">{fmtNum(hoverSample.value)}</span>
      <span class="tip-time">{formatTime(hoverSample.ts)}</span>
    </div>
  {/if}
</div>

<style>
  .chart {
    position: relative;
    width: 100%;
  }
  svg {
    display: block;
    width: 100%;
    height: 160px;
    overflow: visible;
  }
  .grid {
    stroke: var(--border);
    stroke-width: 1;
    vector-effect: non-scaling-stroke;
  }
  .axis {
    fill: var(--text-dim);
    font-size: 9px;
    font-family: var(--mono);
  }
  .area {
    fill: var(--accent);
    opacity: 0.12;
  }
  .line {
    fill: none;
    stroke: var(--accent);
    stroke-width: 1.5;
    stroke-linejoin: round;
    stroke-linecap: round;
    vector-effect: non-scaling-stroke;
  }
  .dot {
    fill: var(--accent);
    opacity: 0.55;
  }
  .dot.solo {
    opacity: 1;
  }
  .cross {
    stroke: var(--text-dim);
    stroke-width: 1;
    stroke-dasharray: 3 3;
    vector-effect: non-scaling-stroke;
  }
  .cursor {
    fill: var(--accent);
    stroke: var(--bg-panel);
    stroke-width: 1.5;
  }
  .tip {
    position: absolute;
    top: 2px;
    transform: translateX(0);
    display: flex;
    flex-direction: column;
    gap: 1px;
    pointer-events: none;
    background: var(--bg-bar);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 3px 7px;
    font-size: 11px;
    white-space: nowrap;
  }
  .tip-val {
    font-family: var(--mono);
    color: var(--text);
    font-weight: 600;
  }
  .tip-time {
    font-family: var(--mono);
    color: var(--text-dim);
    font-size: 10px;
  }
</style>
