// Example decoder plugin — a SOLAR-SITE gateway card.
//
// Invented domain, for illustration. It shows two features together:
//   1. scope: "subtree" — runs when you select a *branch* topic (a site that
//      groups its sensor topics) and receives a snapshot of every descendant's
//      latest value, so the card is stable regardless of receive order.
//   2. html — the plugin returns its own markup and fully owns the right panel,
//      using the app's CSS variables (--accent, --ok, --err, --border,
//      --text-dim, …) to match the theme.
//
// To try it against your own data: copy into the plugins folder (or use the
// manager's Import), then point `topic` at a branch level in your tree and map
// the child paths in decode() to your topics.
//
// Assumed tree under a site, e.g. topic "solar/site/roof-A":
//   solar/site/roof-A/inverter/power     -> "4210"      (W)
//   solar/site/roof-A/inverter/temp      -> "47.5"      (°C)
//   solar/site/roof-A/battery/soc        -> "82"        (%)
//   solar/site/roof-A/meter/yield_kwh    -> "18.4"      (kWh today)
//   solar/site/roof-A/grid/status        -> "online" | "offline"

export default {
  name: "Solar site (example)",
  scope: "subtree",
  topic: "solar/site/+",

  decode(ctx) {
    const num = (k, unit) => {
      const n = ctx.num(k);
      return n === null ? "—" : `${n}${unit}`;
    };
    const online = ctx.get("grid/status") === "online";
    const site = ctx.topic.split("/").pop();

    const tile = (label, value) => `
      <div style="border:1px solid var(--border);border-radius:8px;padding:8px 10px">
        <div style="color:var(--text-dim);font-size:11px;text-transform:uppercase;letter-spacing:.04em">${label}</div>
        <div style="font-size:18px;color:var(--accent);margin-top:2px">${value}</div>
      </div>`;

    return {
      summary: `Solar site · ${site}`,
      html: `
        <div style="display:flex;flex-direction:column;gap:10px">
          <div style="display:flex;align-items:center;gap:8px;font-weight:600">
            <span style="width:9px;height:9px;border-radius:50%;background:var(--${online ? "ok" : "err"})"></span>
            ${online ? "Grid online" : "Grid offline"}
          </div>
          <div style="display:grid;grid-template-columns:repeat(2,1fr);gap:8px">
            ${tile("Power", num("inverter/power", " W"))}
            ${tile("Inverter temp", num("inverter/temp", " °C"))}
            ${tile("Battery", num("battery/soc", " %"))}
            ${tile("Yield today", num("meter/yield_kwh", " kWh"))}
          </div>
        </div>`,
    };
  },
};
