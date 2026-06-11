<script lang="ts">
  // The app icon as inline SVG — mirrors build/appicon.png so the header logo
  // matches the Dock/Finder icon. Scales crisply at any size.
  export let size = 26;
</script>

<svg
  class="logo"
  width={size}
  height={size}
  viewBox="0 0 1024 1024"
  role="img"
  aria-label="MQTT Manager logo"
>
  <defs>
    <linearGradient id="mmLogoGradient" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#5CAAFF" />
      <stop offset="1" stop-color="#1B5FD0" />
    </linearGradient>
  </defs>
  <rect x="0" y="0" width="1024" height="1024" rx="232" ry="232" fill="url(#mmLogoGradient)" />
  <g stroke="#ffffff" stroke-width="34" stroke-linecap="round">
    <line x1="512" y1="360" x2="326" y2="698" />
    <line x1="512" y1="360" x2="512" y2="736" />
    <line x1="512" y1="360" x2="698" y2="698" />
  </g>
  <g fill="#ffffff">
    <circle cx="326" cy="698" r="62" />
    <circle cx="512" cy="736" r="62" />
    <circle cx="698" cy="698" r="62" />
    <circle cx="512" cy="360" r="86" />
  </g>
  <circle cx="512" cy="360" r="60" fill="#5CAAFF" />
  <circle cx="512" cy="360" r="42" fill="#ffffff" />
</svg>

<style>
  .logo {
    display: block;
    flex: 0 0 auto;
  }
</style>
