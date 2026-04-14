const $ = (id) => document.getElementById(id);

const HISTORY_KEY = "aiswitch_account_history_v3";
const HISTORY_LIMIT = 180;
const CHART_ACCOUNT_KEY = "aiswitch_chart_account";
const CHART_METRIC_KEY = "aiswitch_chart_metric";

const PAGE_TITLES = {
  overview: "Overview",
  accounts: "Accounts",
  providers: "Providers",
  analytics: "Analytics",
  routing: "Routing",
  governance: "Governance",
  reliability: "Reliability",
  capabilities: "Capabilities",
};

const CAPABILITY_CATALOG = [
  { id: "c01", title: "Multi-provider account inventory", category: "Foundation", status: "live" },
  { id: "c02", title: "Per-account health scoring", category: "Foundation", status: "live" },
  { id: "c03", title: "Per-account auth expiry tracking", category: "Foundation", status: "live" },
  { id: "c04", title: "Daily budget telemetry", category: "Limits", status: "live" },
  { id: "c05", title: "Weekly budget telemetry", category: "Limits", status: "live" },
  { id: "c06", title: "Monthly budget telemetry", category: "Limits", status: "live" },
  { id: "c07", title: "5-hour request window telemetry", category: "Limits", status: "live" },
  { id: "c08", title: "Rate-limit 5m/hour windows", category: "Limits", status: "live" },
  { id: "c09", title: "Interactive trend chart", category: "Analytics", status: "live" },
  { id: "c10", title: "Historical timeline persistence", category: "Analytics", status: "live" },
  { id: "c11", title: "Metric drilldown by account", category: "Analytics", status: "live" },
  { id: "c12", title: "Provider posture board", category: "Analytics", status: "live" },
  { id: "c13", title: "Risk forecast panel", category: "Analytics", status: "live" },
  { id: "c14", title: "SLA breach engine", category: "Reliability", status: "live" },
  { id: "c15", title: "Critical/warn severity grading", category: "Reliability", status: "live" },
  { id: "c16", title: "One-click account failover", category: "Reliability", status: "live" },
  { id: "c17", title: "Failover incident auto-log", category: "Reliability", status: "live" },
  { id: "c18", title: "Cooldown propagation to profiles", category: "Reliability", status: "live" },
  { id: "c19", title: "Incident timeline console", category: "Reliability", status: "live" },
  { id: "c20", title: "Manual incident automation", category: "Reliability", status: "live" },
  { id: "c21", title: "Route candidate simulation", category: "Routing", status: "live" },
  { id: "c22", title: "Provider preference routing", category: "Routing", status: "live" },
  { id: "c23", title: "Tag-constrained routing", category: "Routing", status: "live" },
  { id: "c24", title: "Protocol-aware routing", category: "Routing", status: "live" },
  { id: "c25", title: "Owner-scoped leasing", category: "Routing", status: "live" },
  { id: "c26", title: "Active lease visibility", category: "Routing", status: "live" },
  { id: "c27", title: "Adapter contract inspection", category: "Routing", status: "live" },
  { id: "c28", title: "Profile CRUD orchestration", category: "Operations", status: "live" },
  { id: "c29", title: "Account telemetry CRUD", category: "Operations", status: "live" },
  { id: "c30", title: "Provider/account filters", category: "Operations", status: "live" },
  { id: "c31", title: "Bulk profile visibility", category: "Operations", status: "live" },
  { id: "c32", title: "Operational notes per account", category: "Operations", status: "live" },
  { id: "c33", title: "Token-based control plane auth", category: "Security", status: "live" },
  { id: "c34", title: "Encrypted secret vault", category: "Security", status: "live" },
  { id: "c35", title: "Secret-to-env bindings", category: "Security", status: "live" },
  { id: "c36", title: "Policy allow/deny providers", category: "Security", status: "live" },
  { id: "c37", title: "Policy auth-method enforcement", category: "Security", status: "live" },
  { id: "c38", title: "Policy budget ceiling", category: "Security", status: "live" },
  { id: "c39", title: "Global dashboard summary endpoint", category: "API", status: "live" },
  { id: "c40", title: "Accounts endpoint suite", category: "API", status: "live" },
  { id: "c41", title: "Account failover endpoint", category: "API", status: "live" },
  { id: "c42", title: "Incidents endpoint suite", category: "API", status: "live" },
  { id: "c43", title: "Leases endpoint suite", category: "API", status: "live" },
  { id: "c44", title: "Metrics export endpoint", category: "API", status: "live" },
  { id: "c45", title: "Multi-page dashboard navigation", category: "UX", status: "live" },
  { id: "c46", title: "Responsive mobile layouts", category: "UX", status: "live" },
  { id: "c47", title: "Toast feedback system", category: "UX", status: "live" },
  { id: "c48", title: "Action-state buttons", category: "UX", status: "live" },
  { id: "c49", title: "Account card action deck", category: "UX", status: "live" },
  { id: "c50", title: "Executive KPI ribbon", category: "UX", status: "live" },
  { id: "c51", title: "Hash-routed deep-link pages", category: "UX", status: "live" },
  { id: "c52", title: "Provider-level aggregation cards", category: "Analytics", status: "live" },
  { id: "c53", title: "Weekly utilization forecasting", category: "Analytics", status: "live" },
  { id: "c54", title: "5-hour burst risk detection", category: "Analytics", status: "live" },
  { id: "c55", title: "Auth expiry countdown warnings", category: "Reliability", status: "live" },
  { id: "c56", title: "Per-page operational workspaces", category: "UX", status: "live" },
  { id: "c57", title: "Provider portfolio indexing", category: "Operations", status: "live" },
  { id: "c58", title: "Alert-driven failover shortcuts", category: "Reliability", status: "live" },
  { id: "c59", title: "Cross-account risk rank ordering", category: "Analytics", status: "live" },
  { id: "c60", title: "Capability matrix governance view", category: "Governance", status: "live" },
];

let authToken = localStorage.getItem("aiswitch_api_token") || "";
let toastTimer = null;
let latestSummary = null;
let historyCache = loadHistoryCache();
let chartTrace = null;

function notify(message, ok = true) {
  const el = $("global-toast");
  if (!el) return;
  el.textContent = message;
  el.className = `toast show ${ok ? "ok" : "err"}`;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => {
    el.className = "toast";
  }, 3200);
}

function setButtonLoading(button, loading, fallbackLabel) {
  if (!button) return;
  if (loading) {
    button.dataset.label = button.textContent;
    button.textContent = "Working...";
    button.disabled = true;
    return;
  }
  button.textContent = button.dataset.label || fallbackLabel || button.textContent;
  button.disabled = false;
}

async function api(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  const hasBody = options.body != null;
  if (hasBody && !("content-type" in headers)) {
    headers["content-type"] = "application/json";
  }
  if (authToken) {
    headers.Authorization = `Bearer ${authToken}`;
  }

  const res = await fetch(path, { headers, ...options });
  const text = await res.text();
  let data = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = { raw: text };
    }
  }
  if (!res.ok) {
    const raw = typeof data?.raw === "string" ? data.raw.replace(/\s+/g, " ").trim() : "";
    let msg = data?.error || raw.slice(0, 180) || `HTTP ${res.status}`;
    if (res.status === 401) {
      msg = "Unauthorized: add your AI Switch API token in the top bar.";
    }
    const err = new Error(msg);
    err.status = res.status;
    throw err;
  }
  return data;
}

function csv(input) {
  if (!input) return [];
  return String(input)
    .split(",")
    .map((v) => v.trim())
    .filter(Boolean);
}

function flash(id, message, ok = true) {
  const el = $(id);
  if (!el) return;
  el.textContent = message;
  el.className = `flash ${ok ? "ok" : "err"}`;
  setTimeout(() => {
    if (el.textContent === message) {
      el.textContent = "";
      el.className = "flash";
    }
  }, 4200);
}

function pretty(id, data) {
  const el = $(id);
  if (!el) return;
  el.textContent = JSON.stringify(data, null, 2);
}

function toNumber(value, fallback = 0) {
  const n = Number(value);
  return Number.isFinite(n) ? n : fallback;
}

function fmtUSD(v) {
  return `$${toNumber(v, 0).toFixed(2)}`;
}

function fmtPct(v) {
  return `${toNumber(v, 0).toFixed(1)}%`;
}

function localDateInput(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (v) => String(v).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function isoFromLocal(value) {
  if (!value) return "";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "";
  return d.toISOString();
}

function fmtDate(iso) {
  if (!iso) return "n/a";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "n/a";
  return d.toLocaleString();
}

function fmtRelative(iso) {
  if (!iso) return "n/a";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "n/a";
  const delta = d.getTime() - Date.now();
  const mins = Math.round(Math.abs(delta) / 60000);
  if (mins < 1) return delta >= 0 ? "now" : "just passed";
  if (mins < 60) return delta >= 0 ? `in ${mins}m` : `${mins}m ago`;
  const hrs = Math.round(mins / 60);
  if (hrs < 48) return delta >= 0 ? `in ${hrs}h` : `${hrs}h ago`;
  const days = Math.round(hrs / 24);
  return delta >= 0 ? `in ${days}d` : `${days}d ago`;
}

function statusClass(status) {
  const s = String(status || "standby").toLowerCase();
  if (["healthy", "degraded", "cooldown", "standby", "disabled"].includes(s)) return s;
  return "standby";
}

function usageBarClass(percent) {
  if (percent >= 90) return "bar bad";
  if (percent >= 75) return "bar warn";
  return "bar";
}

function accountKey(provider, account) {
  return `${String(provider || "").toLowerCase()}::${String(account || "").toLowerCase()}`;
}

function healthPill(entry) {
  const h = entry.health;
  if (!h) return `<span class="pill">no health</span>`;
  const err = toNumber(h.recent_error_rate_percent, 0);
  if (err <= 2) return `<span class="pill ok">${err.toFixed(2)}% err</span>`;
  return `<span class="pill bad">${err.toFixed(2)}% err</span>`;
}

function leasePill(entry) {
  if (!entry.lease) return `<span class="pill">free</span>`;
  return `<span class="pill ok mono">${entry.lease.owner || "assigned"}</span>`;
}

function loadHistoryCache() {
  try {
    const raw = localStorage.getItem(HISTORY_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

function saveHistoryCache() {
  localStorage.setItem(HISTORY_KEY, JSON.stringify(historyCache));
}

function pushHistoryPoint(key, point) {
  if (!historyCache[key]) historyCache[key] = [];
  historyCache[key].push(point);
  if (historyCache[key].length > HISTORY_LIMIT) {
    historyCache[key] = historyCache[key].slice(historyCache[key].length - HISTORY_LIMIT);
  }
}

function recordHistory(accounts) {
  const now = Date.now();
  const aggregate = { t: now, health: 0, daily: 0, weekly: 0, monthly: 0, fivehour: 0, rate5: 0 };
  let count = 0;

  for (const account of accounts) {
    const point = {
      t: now,
      health: toNumber(account.health_score, 0),
      daily: toNumber(account.daily_usage_percent, 0),
      weekly: toNumber(account.weekly_usage_percent, 0),
      monthly: toNumber(account.monthly_usage_percent, 0),
      fivehour: toNumber(account.five_hour_usage_percent, 0),
      rate5: toNumber(account.rate_limit_remaining_5min, 0),
    };
    pushHistoryPoint(accountKey(account.provider, account.account), point);
    aggregate.health += point.health;
    aggregate.daily += point.daily;
    aggregate.weekly += point.weekly;
    aggregate.monthly += point.monthly;
    aggregate.fivehour += point.fivehour;
    aggregate.rate5 += point.rate5;
    count++;
  }

  if (count > 0) {
    aggregate.health /= count;
    aggregate.daily /= count;
    aggregate.weekly /= count;
    aggregate.monthly /= count;
    aggregate.fivehour /= count;
    aggregate.rate5 /= count;
    pushHistoryPoint("__aggregate__", aggregate);
  }

  saveHistoryCache();
}

function metricInfo(metric) {
  switch (metric) {
    case "daily":
      return { label: "Daily usage", unit: "%", color: "#37d6c8" };
    case "weekly":
      return { label: "Weekly usage", unit: "%", color: "#4dc4ff" };
    case "monthly":
      return { label: "Monthly usage", unit: "%", color: "#ffcf86" };
    case "fivehour":
      return { label: "5-hour usage", unit: "%", color: "#ff8fa8" };
    case "rate5":
      return { label: "Rate remaining /5m", unit: "", color: "#9de8bb" };
    case "health":
    default:
      return { label: "Health score", unit: "", color: "#9de8bb" };
  }
}

function getChartSeries() {
  const metric = $("chart-metric-select")?.value || "health";
  const key = $("chart-account-select")?.value || "__aggregate__";
  const source = historyCache[key] || [];
  const series = source.slice(-100).map((p) => ({ t: p.t, v: toNumber(p[metric], 0) }));
  return { metric, key, series };
}

function updateChartAccountOptions(accounts) {
  const select = $("chart-account-select");
  if (!select) return;
  const saved = localStorage.getItem(CHART_ACCOUNT_KEY) || "__aggregate__";
  const keys = ["__aggregate__", ...accounts.map((a) => accountKey(a.provider, a.account))];
  const uniq = [...new Set(keys)].sort();
  select.innerHTML = uniq
    .map((k) => (k === "__aggregate__" ? `<option value="${k}">all accounts</option>` : `<option value="${k}">${k}</option>`))
    .join("");
  select.value = uniq.includes(saved) ? saved : "__aggregate__";
}

function drawTrendChart() {
  const canvas = $("trend-canvas");
  const meta = $("chart-meta");
  if (!canvas || !meta) return;

  const { metric, key, series } = getChartSeries();
  const info = metricInfo(metric);
  const ctx = canvas.getContext("2d");

  const dpr = window.devicePixelRatio || 1;
  const cssW = canvas.clientWidth || 1200;
  const cssH = canvas.clientHeight || 300;
  canvas.width = Math.floor(cssW * dpr);
  canvas.height = Math.floor(cssH * dpr);
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

  ctx.clearRect(0, 0, cssW, cssH);
  ctx.fillStyle = "rgba(7,17,28,0.96)";
  ctx.fillRect(0, 0, cssW, cssH);

  if (series.length < 2) {
    ctx.fillStyle = "#96aac2";
    ctx.font = "13px 'Space Grotesk', sans-serif";
    ctx.fillText("Not enough history yet. Refresh dashboard several times to build trends.", 16, 28);
    meta.textContent = `${info.label}: waiting for timeline.`;
    chartTrace = null;
    return;
  }

  const minRaw = Math.min(...series.map((s) => s.v));
  const maxRaw = Math.max(...series.map((s) => s.v));
  const min = minRaw === maxRaw ? minRaw - 1 : minRaw;
  const max = minRaw === maxRaw ? maxRaw + 1 : maxRaw;

  const pad = { left: 42, right: 14, top: 12, bottom: 28 };
  const w = cssW - pad.left - pad.right;
  const h = cssH - pad.top - pad.bottom;

  ctx.strokeStyle = "rgba(129,188,239,0.2)";
  ctx.lineWidth = 1;
  for (let i = 0; i <= 4; i++) {
    const y = pad.top + (h * i) / 4;
    ctx.beginPath();
    ctx.moveTo(pad.left, y);
    ctx.lineTo(cssW - pad.right, y);
    ctx.stroke();
  }

  const points = series.map((s, idx) => {
    const x = pad.left + (idx * w) / (series.length - 1);
    const ratio = (s.v - min) / (max - min);
    const y = pad.top + h - ratio * h;
    return { ...s, x, y };
  });

  ctx.strokeStyle = info.color;
  ctx.lineWidth = 2.2;
  ctx.beginPath();
  points.forEach((p, i) => {
    if (i === 0) ctx.moveTo(p.x, p.y);
    else ctx.lineTo(p.x, p.y);
  });
  ctx.stroke();

  ctx.fillStyle = info.color;
  for (const p of points) {
    ctx.beginPath();
    ctx.arc(p.x, p.y, 2.4, 0, Math.PI * 2);
    ctx.fill();
  }

  ctx.fillStyle = "#96aac2";
  ctx.font = "11px 'Space Grotesk', sans-serif";
  ctx.fillText(`${max.toFixed(1)}${info.unit}`, 6, pad.top + 8);
  ctx.fillText(`${min.toFixed(1)}${info.unit}`, 6, pad.top + h + 4);

  const latest = points[points.length - 1];
  meta.textContent = `${info.label} (${key}): ${latest.v.toFixed(2)}${info.unit} at ${new Date(latest.t).toLocaleString()}`;

  chartTrace = { points, info };
}

function updateChartHover(clientX) {
  const canvas = $("trend-canvas");
  const meta = $("chart-meta");
  if (!canvas || !meta || !chartTrace?.points?.length) return;

  const rect = canvas.getBoundingClientRect();
  const x = clientX - rect.left;
  let best = chartTrace.points[0];
  let d = Math.abs(best.x - x);
  for (const p of chartTrace.points) {
    const nd = Math.abs(p.x - x);
    if (nd < d) {
      best = p;
      d = nd;
    }
  }
  meta.textContent = `${chartTrace.info.label}: ${best.v.toFixed(2)}${chartTrace.info.unit} at ${new Date(best.t).toLocaleString()}`;
}

function buildSLAAlerts(accounts) {
  const now = Date.now();
  const map = new Map();

  const add = (account, severity, message) => {
    const k = accountKey(account.provider, account.account);
    if (!map.has(k)) {
      map.set(k, { account, severity, notes: [] });
    }
    const item = map.get(k);
    item.notes.push({ severity, message });
    if (severity === "critical") item.severity = "critical";
  };

  for (const account of accounts) {
    const status = statusClass(account.status);
    if (status === "cooldown" || status === "disabled") add(account, "critical", `status ${status}`);
    else if (status === "degraded") add(account, "warn", `status degraded`);

    const d = toNumber(account.daily_usage_percent, 0);
    if (d >= 90) add(account, "critical", `daily usage ${d.toFixed(1)}%`);
    else if (d >= 75) add(account, "warn", `daily usage ${d.toFixed(1)}%`);

    const w = toNumber(account.weekly_usage_percent, 0);
    if (w >= 90) add(account, "critical", `weekly usage ${w.toFixed(1)}%`);
    else if (w >= 75) add(account, "warn", `weekly usage ${w.toFixed(1)}%`);

    const m = toNumber(account.monthly_usage_percent, 0);
    if (m >= 90) add(account, "critical", `monthly usage ${m.toFixed(1)}%`);
    else if (m >= 75) add(account, "warn", `monthly usage ${m.toFixed(1)}%`);

    const f = toNumber(account.five_hour_usage_percent, 0);
    if (f >= 90) add(account, "critical", `5-hour usage ${f.toFixed(1)}%`);
    else if (f >= 75) add(account, "warn", `5-hour usage ${f.toFixed(1)}%`);

    const h = toNumber(account.health_score, 0);
    if (h > 0 && h < 65) add(account, "critical", `health ${h.toFixed(1)}`);
    else if (h > 0 && h < 80) add(account, "warn", `health ${h.toFixed(1)}`);

    const r = toNumber(account.rate_limit_remaining_5min, -1);
    if (r >= 0 && r <= 5) add(account, "critical", `remaining /5m ${r}`);
    else if (r > 5 && r <= 20) add(account, "warn", `remaining /5m ${r}`);

    if (account.auth_expires_at) {
      const exp = new Date(account.auth_expires_at).getTime();
      if (Number.isFinite(exp)) {
        const hours = (exp - now) / 3600000;
        if (hours <= 6) add(account, "critical", `auth expires ${fmtRelative(account.auth_expires_at)}`);
        else if (hours <= 24) add(account, "warn", `auth expires ${fmtRelative(account.auth_expires_at)}`);
      }
    }
  }

  const alerts = Array.from(map.values());
  alerts.sort((a, b) => {
    if (a.severity !== b.severity) return a.severity === "critical" ? -1 : 1;
    return `${a.account.provider}/${a.account.account}`.localeCompare(`${b.account.provider}/${b.account.account}`);
  });
  return alerts;
}

function renderSLAAlerts(accounts) {
  const alerts = buildSLAAlerts(accounts);
  const render = (containerId, limit = 100) => {
    const el = $(containerId);
    if (!el) return;
    const subset = alerts.slice(0, limit);
    if (!subset.length) {
      el.innerHTML = `<div class="alert-card"><div class="alert-title">No active SLA alerts</div><div class="alert-body">All monitored accounts are within thresholds.</div></div>`;
      return;
    }
    el.innerHTML = subset
      .map((entry) => {
        const acc = entry.account;
        const key = accountKey(acc.provider, acc.account);
        return `<article class="alert-card ${entry.severity}">
          <div class="alert-top">
            <div class="alert-title">${acc.provider}/${acc.account}</div>
            <div class="alert-sev">${entry.severity}</div>
          </div>
          <div class="alert-body">${entry.notes.map((n) => n.message).join(" | ")}</div>
          <div class="alert-actions">
            <button class="btn btn-muted btn-compact" type="button" data-load-account="${key}">Edit</button>
            <button class="btn btn-warm btn-compact" type="button" data-failover-account="${acc.provider}::${acc.account}">Failover 15m</button>
          </div>
        </article>`;
      })
      .join("");
  };

  render("sla-alerts");
  render("overview-alerts", 6);
}

function renderProviderBoard(accounts) {
  const providerMap = new Map();
  for (const account of accounts) {
    const key = account.provider || "unknown";
    if (!providerMap.has(key)) {
      providerMap.set(key, {
        provider: key,
        accounts: 0,
        healthy: 0,
        degraded: 0,
        cooldown: 0,
        avgHealth: 0,
        avgWeekly: 0,
      });
    }
    const row = providerMap.get(key);
    row.accounts++;
    const s = statusClass(account.status);
    if (s === "healthy") row.healthy++;
    if (s === "degraded") row.degraded++;
    if (s === "cooldown") row.cooldown++;
    row.avgHealth += toNumber(account.health_score, 0);
    row.avgWeekly += toNumber(account.weekly_usage_percent, 0);
  }

  const providers = Array.from(providerMap.values()).map((p) => {
    p.avgHealth = p.accounts ? p.avgHealth / p.accounts : 0;
    p.avgWeekly = p.accounts ? p.avgWeekly / p.accounts : 0;
    return p;
  });
  providers.sort((a, b) => a.provider.localeCompare(b.provider));

  const render = (id) => {
    const el = $(id);
    if (!el) return;
    if (!providers.length) {
      el.innerHTML = `<article class="provider-card"><h5>No providers</h5><p>Awaiting profile/account data.</p></article>`;
      return;
    }
    el.innerHTML = providers
      .map(
        (p) => `<article class="provider-card">
          <h5>${p.provider}</h5>
          <p>Accounts: ${p.accounts} | Healthy: ${p.healthy} | Degraded: ${p.degraded} | Cooldown: ${p.cooldown}</p>
          <p>Avg Health: ${p.avgHealth.toFixed(1)} | Avg Weekly Usage: ${p.avgWeekly.toFixed(1)}%</p>
        </article>`
      )
      .join("");
  };

  render("providers-board");
  render("overview-provider-board");
}

function renderForecasts(accounts) {
  const el = $("forecast-grid");
  if (!el) return;

  const active = accounts.length;
  const highDaily = accounts.filter((a) => toNumber(a.daily_usage_percent, 0) >= 85).length;
  const highWeekly = accounts.filter((a) => toNumber(a.weekly_usage_percent, 0) >= 85).length;
  const burstRisk = accounts.filter((a) => toNumber(a.five_hour_usage_percent, 0) >= 85).length;
  const expiry24h = accounts.filter((a) => {
    if (!a.auth_expires_at) return false;
    const hours = (new Date(a.auth_expires_at).getTime() - Date.now()) / 3600000;
    return Number.isFinite(hours) && hours > 0 && hours <= 24;
  }).length;
  const degraded = accounts.filter((a) => statusClass(a.status) === "degraded").length;

  const cards = [
    { title: "Active Accounts", value: active, note: "Tracked account objects" },
    { title: "Daily Pressure", value: highDaily, note: ">=85% daily consumption" },
    { title: "Weekly Pressure", value: highWeekly, note: ">=85% weekly consumption" },
    { title: "5h Burst Risk", value: burstRisk, note: ">=85% 5-hour window" },
    { title: "Auth <24h", value: expiry24h, note: "Accounts near expiry" },
    { title: "Degraded Status", value: degraded, note: "Needs intervention" },
  ];

  el.innerHTML = cards
    .map(
      (c) => `<article class="forecast-card"><h5>${c.title}</h5><p>${c.value}</p><p class="hint">${c.note}</p></article>`
    )
    .join("");
}

function renderCapabilities() {
  const list = $("capability-list");
  const stats = $("capability-stats");
  if (!list || !stats) return;

  const byStatus = CAPABILITY_CATALOG.reduce((acc, item) => {
    acc[item.status] = (acc[item.status] || 0) + 1;
    return acc;
  }, {});
  const byCategory = CAPABILITY_CATALOG.reduce((acc, item) => {
    acc[item.category] = (acc[item.category] || 0) + 1;
    return acc;
  }, {});

  list.innerHTML = CAPABILITY_CATALOG.map(
    (item) => `<article class="capability-card">
      <h5>${item.title}</h5>
      <p>${item.category}</p>
      <span class="tag">${item.status}</span>
    </article>`
  ).join("");

  stats.innerHTML = `
    <div class="capability-stat"><h5>Total</h5><p>${CAPABILITY_CATALOG.length}</p></div>
    <div class="capability-stat"><h5>Live</h5><p>${byStatus.live || 0}</p></div>
    <div class="capability-stat"><h5>Categories</h5><p>${Object.keys(byCategory).length}</p></div>
    <div class="capability-stat"><h5>Reliability</h5><p>${byCategory.Reliability || 0}</p></div>
  `;
}

function renderSidebarCounts(summary) {
  const counts = summary.counts || {};
  $("sidebar-profiles").textContent = counts.profiles || 0;
  $("sidebar-accounts").textContent = counts.accounts || 0;
  $("sidebar-providers").textContent = counts.providers || 0;
  $("sidebar-incidents").textContent = counts.incidents || 0;
}

function renderStats(summary) {
  const counts = summary.counts || {};
  const accounts = summary.accounts || [];

  const cards = [
    ["Profiles", counts.profiles || 0],
    ["Accounts", counts.accounts || 0],
    ["Providers", counts.providers || 0],
    ["Active Leases", counts.active_leases || 0],
    ["Healthy Accounts", accounts.filter((a) => statusClass(a.status) === "healthy").length],
    ["SLA Alerts", buildSLAAlerts(accounts).length],
  ];

  const statsEl = $("stats");
  if (statsEl) {
    statsEl.innerHTML = cards.map(([k, v]) => `<div class="stat"><div class="k">${k}</div><div class="v">${v}</div></div>`).join("");
  }

  const providersEl = $("providers-strip");
  if (providersEl) {
    providersEl.innerHTML = Object.entries(summary.providers || {})
      .sort((a, b) => a[0].localeCompare(b[0]))
      .map(([name, count]) => `<span class="provider-pill">${name}<strong>${count}</strong></span>`)
      .join("");
  }

  const updated = summary.time_utc ? new Date(summary.time_utc) : new Date();
  $("as-of").textContent = `Last sync: ${updated.toLocaleString()}`;

  renderSidebarCounts(summary);
}

function updateProviderFilterOptions(accounts) {
  const select = $("account-filter-provider");
  if (!select) return;
  const current = select.value || "all";
  const providers = ["all", ...new Set(accounts.map((a) => a.provider).filter(Boolean))].sort();
  select.innerHTML = providers.map((p) => `<option value="${p}">${p}</option>`).join("");
  if (providers.includes(current)) select.value = current;
}

function filteredAccounts(accounts) {
  const provider = $("account-filter-provider")?.value || "all";
  const status = $("account-filter-status")?.value || "all";
  return accounts.filter((a) => {
    if (provider !== "all" && a.provider !== provider) return false;
    if (status !== "all" && statusClass(a.status) !== status) return false;
    return true;
  });
}

function renderAccounts(summary) {
  const accounts = summary.accounts || [];
  updateProviderFilterOptions(accounts);
  const list = filteredAccounts(accounts);

  const grid = $("accounts-grid");
  if (grid) {
    if (!list.length) {
      grid.innerHTML = `<article class="account-card"><h5>No accounts</h5><p>Adjust filters or add account telemetry.</p></article>`;
    } else {
      grid.innerHTML = list
        .map((a) => {
          const daily = toNumber(a.daily_usage_percent, 0);
          const weekly = toNumber(a.weekly_usage_percent, 0);
          const five = toNumber(a.five_hour_usage_percent, 0);
          return `<article class="account-card">
            <div class="account-top">
              <div>
                <h5>${a.provider}/${a.account}</h5>
                <p>${a.tier || "no tier"} | ${a.auth_method || "auth n/a"}</p>
              </div>
              <span class="status ${statusClass(a.status)}">${statusClass(a.status)}</span>
            </div>

            <div class="metric-pair">
              <div class="metric"><div class="k">Profiles</div><div class="v">${a.profile_count || 0}</div></div>
              <div class="metric"><div class="k">Health</div><div class="v">${toNumber(a.health_score, 0).toFixed(1)}</div></div>
              <div class="metric"><div class="k">Auth Expiry</div><div class="v">${fmtRelative(a.auth_expires_at)}</div></div>
              <div class="metric"><div class="k">5h Remaining</div><div class="v">${toNumber(a.five_hour_remaining, 0)}</div></div>
            </div>

            <div class="usage-line"><div class="k">Daily ${fmtPct(daily)}</div><div class="${usageBarClass(daily)}"><i style="width:${Math.min(100, daily)}%"></i></div></div>
            <div class="usage-line"><div class="k">Weekly ${fmtPct(weekly)}</div><div class="${usageBarClass(weekly)}"><i style="width:${Math.min(100, weekly)}%"></i></div></div>
            <div class="usage-line"><div class="k">5h ${fmtPct(five)}</div><div class="${usageBarClass(five)}"><i style="width:${Math.min(100, five)}%"></i></div></div>

            <div class="account-actions">
              <button class="btn btn-muted btn-compact" type="button" data-load-account="${accountKey(a.provider, a.account)}">Edit</button>
              <button class="btn btn-warm btn-compact" type="button" data-failover-account="${a.provider}::${a.account}">Failover 15m</button>
              <button class="btn btn-danger btn-compact" type="button" data-delete-account="${a.provider}::${a.account}">Delete</button>
            </div>
          </article>`;
        })
        .join("");
    }
  }

  const tbody = $("accounts-table-body");
  if (tbody) {
    tbody.innerHTML = list
      .map(
        (a) => `<tr>
          <td>${a.provider}</td>
          <td class="mono">${a.account}</td>
          <td><span class="status ${statusClass(a.status)}">${statusClass(a.status)}</span></td>
          <td>${a.profile_count || 0}</td>
          <td>${toNumber(a.health_score, 0).toFixed(1)}</td>
          <td title="${fmtDate(a.auth_expires_at)}">${fmtRelative(a.auth_expires_at)}</td>
          <td>${fmtPct(a.five_hour_usage_percent)}</td>
          <td>${fmtPct(a.weekly_usage_percent)}</td>
          <td>${fmtPct(a.monthly_usage_percent)}</td>
          <td>
            <button class="btn btn-muted btn-compact" type="button" data-load-account="${accountKey(a.provider, a.account)}">Edit</button>
            <button class="btn btn-warm btn-compact" type="button" data-failover-account="${a.provider}::${a.account}">Failover</button>
            <button class="btn btn-danger btn-compact" type="button" data-delete-account="${a.provider}::${a.account}">Delete</button>
          </td>
        </tr>`
      )
      .join("");
    if (!tbody.innerHTML.trim()) tbody.innerHTML = `<tr><td colspan="10">No account telemetry available.</td></tr>`;
  }

  renderSLAAlerts(accounts);
  renderProviderBoard(accounts);
  renderForecasts(accounts);
  updateChartAccountOptions(accounts);
  recordHistory(accounts);
  drawTrendChart();
}

function renderProfiles(summary) {
  const tbody = $("profiles-body");
  if (!tbody) return;
  const rows = (summary.profiles || []).map((entry) => {
    const p = entry.profile || {};
    return `<tr>
      <td class="mono">${p.id || "-"}</td>
      <td>${p.provider || "-"}</td>
      <td>${p.frontend || "-"}</td>
      <td>${p.account || "default"}</td>
      <td>${p.priority ?? "-"}</td>
      <td>${leasePill(entry)}</td>
      <td>${healthPill(entry)}</td>
      <td><button class="btn btn-danger btn-compact" type="button" data-delete-profile="${p.id}">Delete</button></td>
    </tr>`;
  });
  tbody.innerHTML = rows.join("") || `<tr><td colspan="8">No profiles configured.</td></tr>`;
}

function findAccountFromSummary(key) {
  if (!latestSummary) return null;
  return (latestSummary.accounts || []).find((a) => accountKey(a.provider, a.account) === key) || null;
}

function populateAccountForm(account) {
  const form = $("account-form");
  if (!form || !account) return;
  form.provider.value = account.provider || "";
  form.account.value = account.account || "";
  form.status.value = statusClass(account.status);
  form.tier.value = account.tier || "";
  form.auth_method.value = account.auth_method || "";
  form.enabled.value = statusClass(account.status) === "disabled" || account.enabled === false ? "false" : "true";

  form.auth_expires_at.value = localDateInput(account.auth_expires_at);
  form.daily_reset_at.value = localDateInput(account.daily_reset_at);
  form.weekly_reset_at.value = localDateInput(account.weekly_reset_at);
  form.monthly_reset_at.value = localDateInput(account.monthly_reset_at);
  form.five_hour_window_reset_at.value = localDateInput(account.five_hour_window_reset_at);
  form.rate_limit_reset_at.value = localDateInput(account.rate_limit_reset_at);

  form.daily_limit_usd.value = toNumber(account.daily_limit_usd, 0);
  form.daily_used_usd.value = toNumber(account.daily_used_usd, 0);
  form.weekly_limit_usd.value = toNumber(account.weekly_limit_usd, 0);
  form.weekly_used_usd.value = toNumber(account.weekly_used_usd, 0);
  form.monthly_limit_usd.value = toNumber(account.monthly_limit_usd, 0);
  form.monthly_used_usd.value = toNumber(account.monthly_used_usd, 0);

  form.five_hour_limit_requests.value = toNumber(account.five_hour_limit_requests, 0);
  form.five_hour_used_requests.value = toNumber(account.five_hour_used_requests, 0);
  form.rate_limit_remaining_5min.value = toNumber(account.rate_limit_remaining_5min, 0);
  form.rate_limit_remaining_hour.value = toNumber(account.rate_limit_remaining_hour, 0);

  form.tags.value = Array.isArray(account.tags) ? account.tags.join(",") : "";
  form.notes.value = account.notes || "";
}

async function runAccountFailover(provider, account, cooldownSeconds = 900, owner = "dashboard", message = "manual failover triggered") {
  return api("v2/accounts/failover", {
    method: "POST",
    body: JSON.stringify({ provider, account, owner, cooldown_seconds: cooldownSeconds, message }),
  });
}

async function refreshSummary() {
  const summary = await api("v2/dashboard/summary");
  latestSummary = summary;
  renderStats(summary);
  renderAccounts(summary);
  renderProfiles(summary);
  pretty("policy-output", summary.policies || []);
  pretty("incident-output", summary.recent_incidents || []);
}

async function refreshSecrets() {
  const secrets = await api("v2/secrets");
  pretty("secrets-output", secrets.items || []);
}

async function refreshAdapters() {
  const [adapters, contract] = await Promise.all([api("v2/adapters"), api("v2/adapters/contract")]);
  pretty("adapters-output", { adapters, contract });
}

async function refreshIncidents() {
  const incidents = await api("v2/incidents?limit=30");
  pretty("incident-output", incidents);
}

async function refreshAll() {
  await Promise.all([refreshSummary(), refreshSecrets(), refreshAdapters(), refreshIncidents()]);
}

function resolvePageFromHash() {
  const raw = (window.location.hash || "#overview").replace(/^#/, "").trim().toLowerCase();
  return PAGE_TITLES[raw] ? raw : "overview";
}

function setPage(page) {
  const safe = PAGE_TITLES[page] ? page : "overview";
  document.querySelectorAll(".page").forEach((el) => {
    el.classList.toggle("active", el.dataset.page === safe);
  });
  document.querySelectorAll(".nav-link").forEach((el) => {
    el.classList.toggle("active", el.dataset.pageLink === safe);
  });
  $("current-page-title").textContent = PAGE_TITLES[safe];
  if ((window.location.hash || "") !== `#${safe}`) {
    window.history.replaceState(null, "", `#${safe}`);
  }
}

function bindPageRouting() {
  document.querySelectorAll(".nav-link").forEach((link) => {
    link.addEventListener("click", (e) => {
      e.preventDefault();
      setPage(link.dataset.pageLink);
    });
  });
  window.addEventListener("hashchange", () => {
    setPage(resolvePageFromHash());
  });
}

function bindEvents() {
  const tokenInput = $("api-token");
  const saveTokenBtn = $("save-token");

  if (tokenInput) tokenInput.value = authToken;

  saveTokenBtn?.addEventListener("click", async () => {
    setButtonLoading(saveTokenBtn, true, "Save Token");
    try {
      authToken = tokenInput.value.trim();
      if (authToken) localStorage.setItem("aiswitch_api_token", authToken);
      else localStorage.removeItem("aiswitch_api_token");
      await refreshAll();
      notify("Token saved and dashboard refreshed.");
    } catch (err) {
      notify(err.message, false);
    } finally {
      setButtonLoading(saveTokenBtn, false, "Save Token");
    }
  });

  $("refresh-all")?.addEventListener("click", async (e) => {
    const btn = e.currentTarget;
    setButtonLoading(btn, true, "Refresh All");
    try {
      await refreshAll();
      notify("All pages refreshed.");
    } catch (err) {
      notify(err.message, false);
    } finally {
      setButtonLoading(btn, false, "Refresh All");
    }
  });

  $("refresh-adapters")?.addEventListener("click", async (e) => {
    const btn = e.currentTarget;
    setButtonLoading(btn, true, "Refresh Adapters");
    try {
      await refreshAdapters();
      notify("Adapter contract refreshed.");
    } catch (err) {
      notify(err.message, false);
    } finally {
      setButtonLoading(btn, false, "Refresh Adapters");
    }
  });

  $("account-filter-provider")?.addEventListener("change", () => latestSummary && renderAccounts(latestSummary));
  $("account-filter-status")?.addEventListener("change", () => latestSummary && renderAccounts(latestSummary));

  $("chart-account-select")?.addEventListener("change", (e) => {
    localStorage.setItem(CHART_ACCOUNT_KEY, e.currentTarget.value);
    drawTrendChart();
  });

  $("chart-metric-select")?.addEventListener("change", (e) => {
    localStorage.setItem(CHART_METRIC_KEY, e.currentTarget.value);
    drawTrendChart();
  });

  const savedMetric = localStorage.getItem(CHART_METRIC_KEY);
  if (savedMetric && $("chart-metric-select")) {
    $("chart-metric-select").value = savedMetric;
  }

  $("trend-canvas")?.addEventListener("mousemove", (e) => updateChartHover(e.clientX));

  $("account-form")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);

    const payload = {
      provider: String(fd.get("provider") || "").trim(),
      account: String(fd.get("account") || "").trim(),
      status: String(fd.get("status") || "").trim(),
      tier: String(fd.get("tier") || "").trim(),
      auth_method: String(fd.get("auth_method") || "").trim(),
      enabled: fd.get("enabled") === "true",
      daily_limit_usd: toNumber(fd.get("daily_limit_usd"), 0),
      daily_used_usd: toNumber(fd.get("daily_used_usd"), 0),
      weekly_limit_usd: toNumber(fd.get("weekly_limit_usd"), 0),
      weekly_used_usd: toNumber(fd.get("weekly_used_usd"), 0),
      monthly_limit_usd: toNumber(fd.get("monthly_limit_usd"), 0),
      monthly_used_usd: toNumber(fd.get("monthly_used_usd"), 0),
      five_hour_limit_requests: toNumber(fd.get("five_hour_limit_requests"), 0),
      five_hour_used_requests: toNumber(fd.get("five_hour_used_requests"), 0),
      rate_limit_remaining_5min: toNumber(fd.get("rate_limit_remaining_5min"), 0),
      rate_limit_remaining_hour: toNumber(fd.get("rate_limit_remaining_hour"), 0),
      tags: csv(fd.get("tags")),
      notes: String(fd.get("notes") || "").trim(),
    };

    const tAuth = isoFromLocal(fd.get("auth_expires_at"));
    const tDaily = isoFromLocal(fd.get("daily_reset_at"));
    const tWeekly = isoFromLocal(fd.get("weekly_reset_at"));
    const tMonthly = isoFromLocal(fd.get("monthly_reset_at"));
    const tFive = isoFromLocal(fd.get("five_hour_window_reset_at"));
    const tRate = isoFromLocal(fd.get("rate_limit_reset_at"));
    if (tAuth) payload.auth_expires_at = tAuth;
    if (tDaily) payload.daily_reset_at = tDaily;
    if (tWeekly) payload.weekly_reset_at = tWeekly;
    if (tMonthly) payload.monthly_reset_at = tMonthly;
    if (tFive) payload.five_hour_window_reset_at = tFive;
    if (tRate) payload.rate_limit_reset_at = tRate;

    const submit = form.querySelector('button[type="submit"]');
    setButtonLoading(submit, true, "Save Account Telemetry");
    try {
      await api("v2/accounts", { method: "POST", body: JSON.stringify(payload) });
      flash("account-flash", `${payload.provider}/${payload.account} saved.`);
      notify(`Account telemetry saved for ${payload.provider}/${payload.account}.`);
      await refreshSummary();
    } catch (err) {
      flash("account-flash", err.message, false);
      notify(err.message, false);
    } finally {
      setButtonLoading(submit, false, "Save Account Telemetry");
    }
  });

  const accountActionHandler = async (e) => {
    const loadKey = e.target?.dataset?.loadAccount;
    if (loadKey) {
      const item = findAccountFromSummary(loadKey);
      if (item) {
        populateAccountForm(item);
        notify(`Loaded ${item.provider}/${item.account} into editor.`);
      }
      return;
    }

    const failoverRaw = e.target?.dataset?.failoverAccount;
    if (failoverRaw) {
      const [provider, account] = failoverRaw.split("::");
      if (!provider || !account) return;
      if (!window.confirm(`Trigger 15-minute failover for ${provider}/${account}?`)) return;
      try {
        const out = await runAccountFailover(provider, account, 900, "dashboard", "manual failover triggered from dashboard card");
        notify(`Failover applied to ${out.affected_profiles} profiles for ${provider}/${account}.`);
        await refreshAll();
      } catch (err) {
        notify(err.message, false);
      }
      return;
    }

    const delRaw = e.target?.dataset?.deleteAccount;
    if (delRaw) {
      const [provider, account] = delRaw.split("::");
      if (!provider || !account) return;
      if (!window.confirm(`Delete account telemetry for ${provider}/${account}?`)) return;
      try {
        await api(`v2/accounts?provider=${encodeURIComponent(provider)}&account=${encodeURIComponent(account)}`, { method: "DELETE" });
        notify(`Deleted telemetry for ${provider}/${account}.`);
        await refreshSummary();
      } catch (err) {
        notify(err.message, false);
      }
    }
  };

  $("accounts-grid")?.addEventListener("click", accountActionHandler);
  $("accounts-table-body")?.addEventListener("click", accountActionHandler);
  $("sla-alerts")?.addEventListener("click", accountActionHandler);
  $("overview-alerts")?.addEventListener("click", accountActionHandler);

  $("profile-form")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const payload = {
      id: fd.get("id"),
      provider: fd.get("provider"),
      frontend: fd.get("frontend"),
      auth_method: fd.get("auth_method"),
      protocol: fd.get("protocol"),
      account: fd.get("account"),
      priority: toNumber(fd.get("priority"), 0),
      enabled: fd.get("enabled") === "true",
      budget_daily_usd: toNumber(fd.get("budget_daily_usd"), 0),
      tags: csv(fd.get("tags")),
      owner_scopes: csv(fd.get("owner_scopes")),
    };

    const submit = form.querySelector('button[type="submit"]');
    setButtonLoading(submit, true, "Save Profile");
    try {
      await api("v2/profiles", { method: "POST", body: JSON.stringify(payload) });
      flash("profile-flash", `Profile ${payload.id} saved.`);
      notify(`Profile ${payload.id} saved.`);
      form.reset();
      await refreshSummary();
    } catch (err) {
      flash("profile-flash", err.message, false);
      notify(err.message, false);
    } finally {
      setButtonLoading(submit, false, "Save Profile");
    }
  });

  $("profiles-body")?.addEventListener("click", async (e) => {
    const id = e.target?.dataset?.deleteProfile;
    if (!id) return;
    if (!window.confirm(`Delete profile ${id}?`)) return;
    try {
      await api(`v2/profiles?id=${encodeURIComponent(id)}`, { method: "DELETE" });
      notify(`Profile ${id} deleted.`);
      await refreshSummary();
    } catch (err) {
      notify(err.message, false);
    }
  });

  $("route-form")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const payload = {
      frontend: fd.get("frontend") || "",
      task_class: fd.get("task_class") || "coding",
      required_protocol: fd.get("required_protocol") || "",
      preferred_providers: csv(fd.get("preferred_providers")),
      require_tags: csv(fd.get("require_tags")),
      owner: fd.get("owner") || "dashboard",
    };
    const submit = form.querySelector('button[type="submit"]');
    setButtonLoading(submit, true, "Simulate Route");
    try {
      const out = await api("v2/route/candidates", { method: "POST", body: JSON.stringify(payload) });
      pretty("route-output", out);
    } catch (err) {
      pretty("route-output", { error: err.message });
      notify(err.message, false);
    } finally {
      setButtonLoading(submit, false, "Simulate Route");
    }
  });

  $("secret-form")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const submit = form.querySelector('button[type="submit"]');
    setButtonLoading(submit, true, "Store Secret");
    try {
      await api("v2/secrets", { method: "POST", body: JSON.stringify({ name: fd.get("name"), value: fd.get("value") }) });
      flash("secret-flash", `Secret ${fd.get("name")} stored.`);
      notify(`Secret ${fd.get("name")} stored.`);
      form.reset();
      await refreshSecrets();
    } catch (err) {
      flash("secret-flash", err.message, false);
      notify(err.message, false);
    } finally {
      setButtonLoading(submit, false, "Store Secret");
    }
  });

  $("binding-form")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const submit = form.querySelector('button[type="submit"]');
    setButtonLoading(submit, true, "Bind Secret");
    try {
      await api("v2/secret-bindings", {
        method: "POST",
        body: JSON.stringify({ profile_id: fd.get("profile_id"), env_var: fd.get("env_var"), secret_key: fd.get("secret_key") }),
      });
      flash("secret-flash", "Binding saved.");
      notify("Secret binding saved.");
      form.reset();
      await refreshSummary();
    } catch (err) {
      flash("secret-flash", err.message, false);
      notify(err.message, false);
    } finally {
      setButtonLoading(submit, false, "Bind Secret");
    }
  });

  $("policy-form")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const payload = {
      name: fd.get("name"),
      priority: toNumber(fd.get("priority"), 100),
      frontends: csv(fd.get("frontends")),
      task_classes: csv(fd.get("task_classes")),
      allow_providers: csv(fd.get("allow_providers")),
      deny_providers: csv(fd.get("deny_providers")),
      require_any_tag: csv(fd.get("require_any_tag")),
      require_auth_methods: csv(fd.get("require_auth_methods")),
      max_budget_daily_usd: toNumber(fd.get("max_budget_daily_usd"), 0),
    };
    const submit = form.querySelector('button[type="submit"]');
    setButtonLoading(submit, true, "Save Policy");
    try {
      await api("v2/policies", { method: "POST", body: JSON.stringify(payload) });
      flash("policy-flash", `Policy ${payload.name} saved.`);
      notify(`Policy ${payload.name} saved.`);
      form.reset();
      await refreshSummary();
    } catch (err) {
      flash("policy-flash", err.message, false);
      notify(err.message, false);
    } finally {
      setButtonLoading(submit, false, "Save Policy");
    }
  });

  $("incident-form")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const payload = {
      profile_id: fd.get("profile_id"),
      kind: fd.get("kind"),
      message: fd.get("message"),
      owner: fd.get("owner"),
      cooldown_seconds: toNumber(fd.get("cooldown_seconds"), 0),
    };
    const submit = form.querySelector('button[type="submit"]');
    setButtonLoading(submit, true, "Record Incident");
    try {
      await api("v2/incidents", { method: "POST", body: JSON.stringify(payload) });
      flash("incident-flash", "Incident recorded and cooldown applied.");
      notify("Incident recorded.");
      await refreshAll();
    } catch (err) {
      flash("incident-flash", err.message, false);
      notify(err.message, false);
    } finally {
      setButtonLoading(submit, false, "Record Incident");
    }
  });

  $("failover-form")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const payload = {
      provider: String(fd.get("provider") || "").trim(),
      account: String(fd.get("account") || "").trim(),
      cooldown: toNumber(fd.get("cooldown_seconds"), 900),
      owner: String(fd.get("owner") || "dashboard").trim() || "dashboard",
      message: String(fd.get("message") || "").trim() || "manual failover triggered from reliability console",
    };
    const submit = form.querySelector('button[type="submit"]');
    setButtonLoading(submit, true, "Trigger Failover");
    try {
      const out = await runAccountFailover(payload.provider, payload.account, payload.cooldown, payload.owner, payload.message);
      pretty("failover-output", out);
      flash("failover-flash", `Failover applied to ${out.affected_profiles} profiles.`);
      notify(`Failover executed for ${payload.provider}/${payload.account}.`);
      await refreshAll();
    } catch (err) {
      pretty("failover-output", { error: err.message });
      flash("failover-flash", err.message, false);
      notify(err.message, false);
    } finally {
      setButtonLoading(submit, false, "Trigger Failover");
    }
  });
}

async function init() {
  bindPageRouting();
  bindEvents();
  renderCapabilities();
  setPage(resolvePageFromHash());
  try {
    await refreshAll();
    notify("Dashboard connected.");
  } catch (err) {
    if (err?.status === 401) {
      $("as-of").textContent = "Authentication required. Add API token to load data.";
      notify("Add your AI Switch API token to load live telemetry.");
      return;
    }
    notify(err.message, false);
  }
}

init();
