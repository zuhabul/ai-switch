const $ = (id) => document.getElementById(id);

const HISTORY_KEY = "aiswitch_account_history_v2";
const HISTORY_LIMIT = 140;
const CHART_ACCOUNT_KEY = "aiswitch_chart_account";
const CHART_METRIC_KEY = "aiswitch_chart_metric";

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
    const msg = data?.error || raw.slice(0, 180) || `HTTP ${res.status}`;
    throw new Error(msg);
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
  if (Number.isFinite(n)) return n;
  return fallback;
}

function fmtUSD(value) {
  const n = Number(value || 0);
  return `$${n.toFixed(2)}`;
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
  const errPct = toNumber(h.recent_error_rate_percent, 0);
  if (errPct <= 2) {
    return `<span class="pill ok">${errPct.toFixed(2)}% err</span>`;
  }
  return `<span class="pill bad">${errPct.toFixed(2)}% err</span>`;
}

function leasePill(entry) {
  if (!entry.lease) return `<span class="pill">free</span>`;
  return `<span class="pill ok mono">${entry.lease.owner || "assigned"}</span>`;
}

function accountUsageText(used, limit, remaining) {
  if (!limit || limit <= 0) {
    return `${fmtUSD(used)} / uncapped`;
  }
  return `${fmtUSD(used)} / ${fmtUSD(limit)} (${fmtUSD(remaining)} left)`;
}

function loadHistoryCache() {
  try {
    const raw = localStorage.getItem(HISTORY_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") return {};
    return parsed;
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
  let aggHealth = 0;
  let aggDaily = 0;
  let aggMonthly = 0;
  let aggRate5 = 0;
  let count = 0;

  for (const account of accounts) {
    const point = {
      t: now,
      health: toNumber(account.health_score, 0),
      daily: toNumber(account.daily_usage_percent, 0),
      monthly: toNumber(account.monthly_usage_percent, 0),
      rate5: toNumber(account.rate_limit_remaining_5min, 0),
    };
    pushHistoryPoint(accountKey(account.provider, account.account), point);
    aggHealth += point.health;
    aggDaily += point.daily;
    aggMonthly += point.monthly;
    aggRate5 += point.rate5;
    count++;
  }

  if (count > 0) {
    pushHistoryPoint("__aggregate__", {
      t: now,
      health: aggHealth / count,
      daily: aggDaily / count,
      monthly: aggMonthly / count,
      rate5: aggRate5 / count,
    });
  }

  saveHistoryCache();
}

function metricInfo(metric) {
  switch (metric) {
    case "daily":
      return { label: "Daily Usage", unit: "%", color: "#31e8c0" };
    case "monthly":
      return { label: "Monthly Usage", unit: "%", color: "#ffc978" };
    case "rate5":
      return { label: "Rate Remaining /5m", unit: "", color: "#3fd8ff" };
    case "health":
    default:
      return { label: "Health Score", unit: "", color: "#9ff5ba" };
  }
}

function getChartSeries() {
  const select = $("chart-account-select");
  const metricSelect = $("chart-metric-select");
  const key = select?.value || "__aggregate__";
  const metric = metricSelect?.value || "health";
  const source = historyCache[key] || [];
  const series = source.slice(-80).map((point) => ({
    t: point.t,
    v: toNumber(point[metric], 0),
  }));
  return { key, metric, series };
}

function updateChartAccountOptions(accounts) {
  const select = $("chart-account-select");
  if (!select) return;

  const current = localStorage.getItem(CHART_ACCOUNT_KEY) || select.value || "__aggregate__";
  const options = ["__aggregate__", ...accounts.map((a) => accountKey(a.provider, a.account))];
  const unique = [...new Set(options)].sort();

  select.innerHTML = unique
    .map((key) => {
      if (key === "__aggregate__") return `<option value="__aggregate__">all accounts</option>`;
      return `<option value="${key}">${key}</option>`;
    })
    .join("");

  select.value = unique.includes(current) ? current : "__aggregate__";
  localStorage.setItem(CHART_ACCOUNT_KEY, select.value);
}

function drawTrendChart() {
  const canvas = $("trend-canvas");
  const meta = $("chart-meta");
  if (!canvas || !meta) return;

  const { key, metric, series } = getChartSeries();
  const info = metricInfo(metric);

  const ctx = canvas.getContext("2d");
  const dpr = window.devicePixelRatio || 1;
  const cssW = canvas.clientWidth || 860;
  const cssH = canvas.clientHeight || 260;
  canvas.width = Math.floor(cssW * dpr);
  canvas.height = Math.floor(cssH * dpr);
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

  ctx.clearRect(0, 0, cssW, cssH);
  ctx.fillStyle = "rgba(7,20,34,0.96)";
  ctx.fillRect(0, 0, cssW, cssH);

  if (series.length < 2) {
    ctx.fillStyle = "#9bb5cc";
    ctx.font = "13px Sora";
    ctx.fillText("Not enough history yet. Keep refreshing to build a timeline.", 16, 28);
    meta.textContent = `${info.label}: waiting for timeline points.`;
    chartTrace = null;
    return;
  }

  const minRaw = Math.min(...series.map((p) => p.v));
  const maxRaw = Math.max(...series.map((p) => p.v));
  const min = minRaw === maxRaw ? minRaw - 1 : minRaw;
  const max = minRaw === maxRaw ? maxRaw + 1 : maxRaw;
  const pad = { left: 42, right: 14, top: 12, bottom: 28 };
  const w = cssW - pad.left - pad.right;
  const h = cssH - pad.top - pad.bottom;

  ctx.strokeStyle = "rgba(112, 192, 242, 0.18)";
  ctx.lineWidth = 1;
  for (let i = 0; i <= 4; i++) {
    const y = pad.top + (h * i) / 4;
    ctx.beginPath();
    ctx.moveTo(pad.left, y);
    ctx.lineTo(cssW - pad.right, y);
    ctx.stroke();
  }

  const points = series.map((p, idx) => {
    const x = pad.left + (idx * w) / (series.length - 1);
    const ratio = (p.v - min) / (max - min);
    const y = pad.top + h - ratio * h;
    return { ...p, x, y };
  });

  ctx.lineWidth = 2.2;
  ctx.strokeStyle = info.color;
  ctx.beginPath();
  points.forEach((point, i) => {
    if (i === 0) ctx.moveTo(point.x, point.y);
    else ctx.lineTo(point.x, point.y);
  });
  ctx.stroke();

  ctx.fillStyle = info.color;
  points.forEach((point) => {
    ctx.beginPath();
    ctx.arc(point.x, point.y, 2.3, 0, Math.PI * 2);
    ctx.fill();
  });

  ctx.fillStyle = "#9bb5cc";
  ctx.font = "12px Sora";
  ctx.fillText(`${max.toFixed(1)}${info.unit}`, 6, pad.top + 6);
  ctx.fillText(`${min.toFixed(1)}${info.unit}`, 6, pad.top + h + 4);

  const latest = points[points.length - 1];
  meta.textContent = `${info.label} (${key}): ${latest.v.toFixed(2)}${info.unit} | ${new Date(latest.t).toLocaleTimeString()}`;

  chartTrace = { points, info, pad, w, h, cssW, cssH };
}

function updateChartHover(clientX) {
  const canvas = $("trend-canvas");
  const meta = $("chart-meta");
  if (!canvas || !meta || !chartTrace || !chartTrace.points?.length) return;

  const rect = canvas.getBoundingClientRect();
  const x = clientX - rect.left;
  let best = chartTrace.points[0];
  let bestDistance = Math.abs(best.x - x);
  for (const point of chartTrace.points) {
    const dist = Math.abs(point.x - x);
    if (dist < bestDistance) {
      best = point;
      bestDistance = dist;
    }
  }
  const ts = new Date(best.t).toLocaleString();
  meta.textContent = `${chartTrace.info.label}: ${best.v.toFixed(2)}${chartTrace.info.unit} at ${ts}`;
}

function accountAlertSeverity(alerts) {
  if (alerts.some((a) => a.severity === "critical")) return "critical";
  return "warn";
}

function buildSLAAlerts(accounts) {
  const now = Date.now();
  const grouped = new Map();

  const add = (account, severity, message) => {
    const key = accountKey(account.provider, account.account);
    if (!grouped.has(key)) {
      grouped.set(key, {
        account,
        messages: [],
        severity,
      });
    }
    const entry = grouped.get(key);
    entry.messages.push({ severity, message });
    if (severity === "critical") entry.severity = "critical";
  };

  for (const account of accounts) {
    const status = statusClass(account.status);
    if (status === "cooldown" || status === "disabled") {
      add(account, "critical", `status is ${status}`);
    } else if (status === "degraded") {
      add(account, "warn", "status is degraded");
    }

    const daily = toNumber(account.daily_usage_percent, 0);
    if (daily >= 90) add(account, "critical", `daily usage ${daily.toFixed(1)}%`);
    else if (daily >= 75) add(account, "warn", `daily usage ${daily.toFixed(1)}%`);

    const monthly = toNumber(account.monthly_usage_percent, 0);
    if (monthly >= 90) add(account, "critical", `monthly usage ${monthly.toFixed(1)}%`);
    else if (monthly >= 75) add(account, "warn", `monthly usage ${monthly.toFixed(1)}%`);

    const health = toNumber(account.health_score, 0);
    if (health > 0 && health < 65) add(account, "critical", `health score ${health.toFixed(1)}`);
    else if (health > 0 && health < 80) add(account, "warn", `health score ${health.toFixed(1)}`);

    const rate5 = toNumber(account.rate_limit_remaining_5min, -1);
    if (rate5 >= 0 && rate5 <= 5) add(account, "critical", `remaining /5m ${rate5}`);
    else if (rate5 > 5 && rate5 <= 20) add(account, "warn", `remaining /5m ${rate5}`);

    if (account.auth_expires_at) {
      const exp = new Date(account.auth_expires_at).getTime();
      if (Number.isFinite(exp)) {
        const hours = (exp - now) / 3600000;
        if (hours <= 6) add(account, "critical", `auth expires ${fmtRelative(account.auth_expires_at)}`);
        else if (hours <= 24) add(account, "warn", `auth expires ${fmtRelative(account.auth_expires_at)}`);
      }
    }
  }

  const alerts = Array.from(grouped.values()).map((item) => ({
    ...item,
    severity: accountAlertSeverity(item.messages),
  }));

  alerts.sort((a, b) => {
    if (a.severity !== b.severity) return a.severity === "critical" ? -1 : 1;
    return `${a.account.provider}/${a.account.account}`.localeCompare(`${b.account.provider}/${b.account.account}`);
  });
  return alerts;
}

function renderSLAAlerts(accounts) {
  const wrap = $("sla-alerts");
  if (!wrap) return;
  const alerts = buildSLAAlerts(accounts);
  if (!alerts.length) {
    wrap.innerHTML = `<div class="alert-card"><div class="alert-title">No active SLA alerts</div><div class="alert-body">All tracked accounts are currently within configured risk thresholds.</div></div>`;
    return;
  }

  wrap.innerHTML = alerts
    .map((entry) => {
      const account = entry.account;
      const key = accountKey(account.provider, account.account);
      return `<article class="alert-card ${entry.severity}">
        <div class="alert-top">
          <div class="alert-title">${account.provider}/${account.account}</div>
          <div class="alert-sev">${entry.severity}</div>
        </div>
        <div class="alert-body">${entry.messages.map((m) => m.message).join(" | ")}</div>
        <div class="alert-actions">
          <button class="btn btn-muted btn-compact" type="button" data-load-account="${key}">Edit</button>
          <button class="btn btn-warm btn-compact" type="button" data-failover-account="${account.provider}::${account.account}">Failover 15m</button>
        </div>
      </article>`;
    })
    .join("");
}

function renderStats(summary) {
  const counts = summary.counts || {};
  const accounts = summary.accounts || [];
  const healthyAccounts = accounts.filter((a) => statusClass(a.status) === "healthy").length;
  const highPressure = accounts.filter((a) => toNumber(a.daily_usage_percent, 0) >= 80).length;
  const cards = [
    ["Profiles", counts.profiles || 0],
    ["Accounts", counts.accounts || 0],
    ["Providers", counts.providers || 0],
    ["Active Leases", counts.active_leases || 0],
    ["Healthy Accounts", healthyAccounts],
    ["High Pressure", highPressure],
  ];
  const stats = $("stats");
  if (stats) {
    stats.innerHTML = cards
      .map(([k, v]) => `<div class="stat"><div class="k">${k}</div><div class="v">${v}</div></div>`)
      .join("");
  }
  const updated = summary.time_utc ? new Date(summary.time_utc) : new Date();
  $("as-of").textContent = updated.toLocaleString();

  const providers = Object.entries(summary.providers || {}).sort((a, b) => a[0].localeCompare(b[0]));
  const providerStrip = $("providers-strip");
  if (providerStrip) {
    providerStrip.innerHTML = providers
      .map(([name, count]) => `<span class="provider-pill">${name}<strong>${count}</strong></span>`)
      .join("");
  }
}

function updateProviderFilterOptions(accounts) {
  const select = $("account-filter-provider");
  if (!select) return;
  const current = select.value || "all";
  const providers = ["all", ...new Set(accounts.map((a) => a.provider).filter(Boolean))].sort();
  select.innerHTML = providers
    .map((provider) => `<option value="${provider}">${provider}</option>`)
    .join("");
  if (providers.includes(current)) {
    select.value = current;
  }
}

function filteredAccounts(accounts) {
  const provider = $("account-filter-provider")?.value || "all";
  const status = $("account-filter-status")?.value || "all";
  return accounts.filter((account) => {
    if (provider !== "all" && account.provider !== provider) return false;
    if (status !== "all" && statusClass(account.status) !== status) return false;
    return true;
  });
}

function renderAccounts(summary) {
  const accounts = summary.accounts || [];
  updateProviderFilterOptions(accounts);
  const list = filteredAccounts(accounts);

  const grid = $("accounts-grid");
  if (!grid) return;

  if (!list.length) {
    grid.innerHTML = `<article class="account-card"><strong>No accounts found for current filter.</strong><span class="subhead">Create telemetry in the editor to onboard a provider account.</span></article>`;
  } else {
    grid.innerHTML = list
      .map((account) => {
        const status = statusClass(account.status);
        const dailyUsage = toNumber(account.daily_usage_percent, 0);
        const monthlyUsage = toNumber(account.monthly_usage_percent, 0);
        return `
          <article class="account-card" data-account-key="${accountKey(account.provider, account.account)}">
            <div class="account-head">
              <div class="account-title">
                <strong>${account.provider}/${account.account}</strong>
                <span>${account.tier || "no tier"} | ${account.auth_method || "auth n/a"}</span>
              </div>
              <span class="status ${status}">${status}</span>
            </div>

            <div class="account-metrics">
              <div class="mini"><div class="k">Profiles</div><div class="v">${account.profile_count || 0}</div></div>
              <div class="mini"><div class="k">Health Score</div><div class="v">${toNumber(account.health_score, 0).toFixed(1)}</div></div>
              <div class="mini"><div class="k">Auth Expiry</div><div class="v">${fmtRelative(account.auth_expires_at)}</div></div>
              <div class="mini"><div class="k">Rate /5m</div><div class="v">${toNumber(account.rate_limit_remaining_5min, 0)}</div></div>
            </div>

            <div class="usage">
              <div class="k">Daily Usage ${dailyUsage.toFixed(1)}%</div>
              <div class="${usageBarClass(dailyUsage)}"><i style="width:${Math.min(100, dailyUsage)}%"></i></div>
              <div class="subhead">${accountUsageText(account.daily_used_usd, account.daily_limit_usd, account.daily_remaining_usd)}</div>
            </div>

            <div class="usage">
              <div class="k">Monthly Usage ${monthlyUsage.toFixed(1)}%</div>
              <div class="${usageBarClass(monthlyUsage)}"><i style="width:${Math.min(100, monthlyUsage)}%"></i></div>
              <div class="subhead">${accountUsageText(account.monthly_used_usd, account.monthly_limit_usd, account.monthly_remaining_usd)}</div>
            </div>

            <div class="account-actions">
              <button class="btn btn-muted btn-compact" type="button" data-load-account="${accountKey(account.provider, account.account)}">Edit</button>
              <button class="btn btn-warm btn-compact" type="button" data-failover-account="${account.provider}::${account.account}">Failover 15m</button>
              <button class="btn btn-danger btn-compact" type="button" data-delete-account="${account.provider}::${account.account}">Delete</button>
            </div>
          </article>`;
      })
      .join("");
  }

  const tableBody = $("accounts-table-body");
  if (tableBody) {
    tableBody.innerHTML = list
      .map((account) => {
        const status = statusClass(account.status);
        return `<tr>
          <td>${account.provider}</td>
          <td class="mono">${account.account}</td>
          <td><span class="status ${status}">${status}</span></td>
          <td>${account.profile_count || 0}</td>
          <td>${toNumber(account.health_score, 0).toFixed(1)}</td>
          <td title="${fmtDate(account.auth_expires_at)}">${fmtRelative(account.auth_expires_at)}</td>
          <td>${toNumber(account.daily_usage_percent, 0).toFixed(1)}%</td>
          <td>${toNumber(account.monthly_usage_percent, 0).toFixed(1)}%</td>
          <td>${toNumber(account.rate_limit_remaining_5min, 0)} / ${toNumber(account.rate_limit_remaining_hour, 0)}</td>
          <td>
            <button class="btn btn-muted btn-compact" type="button" data-load-account="${accountKey(account.provider, account.account)}">Edit</button>
            <button class="btn btn-warm btn-compact" type="button" data-failover-account="${account.provider}::${account.account}">Failover</button>
            <button class="btn btn-danger btn-compact" type="button" data-delete-account="${account.provider}::${account.account}">Delete</button>
          </td>
        </tr>`;
      })
      .join("");

    if (!tableBody.innerHTML.trim()) {
      tableBody.innerHTML = `<tr><td colspan="10">No account telemetry available.</td></tr>`;
    }
  }

  renderSLAAlerts(accounts);
  updateChartAccountOptions(accounts);
  recordHistory(accounts);
  drawTrendChart();
}

function renderProfiles(summary) {
  const body = $("profiles-body");
  if (!body) return;
  const rows = (summary.profiles || []).map((entry) => {
    const profile = entry.profile || {};
    return `<tr>
      <td class="mono">${profile.id || "-"}</td>
      <td>${profile.provider || "-"}</td>
      <td>${profile.frontend || "-"}</td>
      <td>${profile.account || "default"}</td>
      <td>${profile.priority ?? "-"}</td>
      <td>${leasePill(entry)}</td>
      <td>${healthPill(entry)}</td>
      <td><button class="btn btn-danger btn-compact" type="button" data-delete-profile="${profile.id}">Delete</button></td>
    </tr>`;
  });
  body.innerHTML = rows.join("") || `<tr><td colspan="8">No profiles configured.</td></tr>`;
}

function findAccountFromSummary(key) {
  if (!latestSummary) return null;
  const items = latestSummary.accounts || [];
  return items.find((item) => accountKey(item.provider, item.account) === key) || null;
}

function populateAccountForm(account) {
  const form = $("account-form");
  if (!form || !account) return;
  form.provider.value = account.provider || "";
  form.account.value = account.account || "";
  form.status.value = statusClass(account.status);
  form.tier.value = account.tier || "";
  form.auth_method.value = account.auth_method || "";
  form.enabled.value = statusClass(account.status) === "disabled" ? "false" : "true";
  form.auth_expires_at.value = localDateInput(account.auth_expires_at);
  form.rate_limit_reset_at.value = localDateInput(account.rate_limit_reset_at);
  form.daily_limit_usd.value = toNumber(account.daily_limit_usd, 0);
  form.daily_used_usd.value = toNumber(account.daily_used_usd, 0);
  form.daily_reset_at.value = localDateInput(account.daily_reset_at);
  form.monthly_limit_usd.value = toNumber(account.monthly_limit_usd, 0);
  form.monthly_used_usd.value = toNumber(account.monthly_used_usd, 0);
  form.monthly_reset_at.value = localDateInput(account.monthly_reset_at);
  form.rate_limit_remaining_5min.value = toNumber(account.rate_limit_remaining_5min, 0);
  form.rate_limit_remaining_hour.value = toNumber(account.rate_limit_remaining_hour, 0);
  form.tags.value = Array.isArray(account.tags) ? account.tags.join(",") : "";
  form.notes.value = account.notes || "";
}

async function runAccountFailover(provider, account, cooldownSeconds = 900) {
  const payload = {
    provider,
    account,
    owner: "dashboard",
    cooldown_seconds: cooldownSeconds,
    message: "manual failover triggered from account control tower",
  };
  const out = await api("v2/accounts/failover", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  return out;
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

function setActiveSection(sectionId) {
  document.querySelectorAll(".section-nav a").forEach((link) => {
    const isActive = link.getAttribute("href") === `#${sectionId}`;
    link.classList.toggle("is-active", isActive);
  });
}

function bindSectionObserver() {
  const sections = document.querySelectorAll("section[id]");
  if (!sections.length || !("IntersectionObserver" in window)) return;
  const observer = new IntersectionObserver(
    (entries) => {
      const active = entries
        .filter((entry) => entry.isIntersecting)
        .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];
      if (active?.target?.id) {
        setActiveSection(active.target.id);
      }
    },
    { threshold: [0.2, 0.4, 0.65], rootMargin: "-15% 0px -45% 0px" }
  );
  sections.forEach((section) => observer.observe(section));
}

function bindEvents() {
  const tokenInput = $("api-token");
  const saveTokenBtn = $("save-token");

  if (tokenInput) tokenInput.value = authToken;

  saveTokenBtn?.addEventListener("click", async () => {
    setButtonLoading(saveTokenBtn, true, "Save Token");
    try {
      authToken = tokenInput.value.trim();
      if (authToken) {
        localStorage.setItem("aiswitch_api_token", authToken);
      } else {
        localStorage.removeItem("aiswitch_api_token");
      }
      await refreshAll();
      notify("Token saved and dashboard refreshed.");
    } catch (err) {
      notify(err.message, false);
    } finally {
      setButtonLoading(saveTokenBtn, false, "Save Token");
    }
  });

  $("refresh-all")?.addEventListener("click", async (e) => {
    const button = e.currentTarget;
    setButtonLoading(button, true, "Refresh All");
    try {
      await refreshAll();
      notify("All sections refreshed.");
    } catch (err) {
      notify(err.message, false);
    } finally {
      setButtonLoading(button, false, "Refresh All");
    }
  });

  $("refresh-adapters")?.addEventListener("click", async (e) => {
    const button = e.currentTarget;
    setButtonLoading(button, true, "Refresh Adapters");
    try {
      await refreshAdapters();
      notify("Adapter contract refreshed.");
    } catch (err) {
      notify(err.message, false);
    } finally {
      setButtonLoading(button, false, "Refresh Adapters");
    }
  });

  $("account-filter-provider")?.addEventListener("change", () => {
    if (latestSummary) renderAccounts(latestSummary);
  });
  $("account-filter-status")?.addEventListener("change", () => {
    if (latestSummary) renderAccounts(latestSummary);
  });

  $("chart-account-select")?.addEventListener("change", (e) => {
    localStorage.setItem(CHART_ACCOUNT_KEY, e.currentTarget.value);
    drawTrendChart();
  });
  $("chart-metric-select")?.addEventListener("change", (e) => {
    localStorage.setItem(CHART_METRIC_KEY, e.currentTarget.value);
    drawTrendChart();
  });

  const chartMetric = localStorage.getItem(CHART_METRIC_KEY);
  if (chartMetric && $("chart-metric-select")) {
    $("chart-metric-select").value = chartMetric;
  }

  $("trend-canvas")?.addEventListener("mousemove", (e) => updateChartHover(e.clientX));
  $("trend-canvas")?.addEventListener("mouseleave", () => {
    if (latestSummary) {
      const { metric, key } = getChartSeries();
      const info = metricInfo(metric);
      $("chart-meta").textContent = `${info.label} (${key}) trend ready.`;
    }
  });

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
      daily_limit_usd: toNumber(fd.get("daily_limit_usd"), 0),
      daily_used_usd: toNumber(fd.get("daily_used_usd"), 0),
      monthly_limit_usd: toNumber(fd.get("monthly_limit_usd"), 0),
      monthly_used_usd: toNumber(fd.get("monthly_used_usd"), 0),
      rate_limit_remaining_5min: toNumber(fd.get("rate_limit_remaining_5min"), 0),
      rate_limit_remaining_hour: toNumber(fd.get("rate_limit_remaining_hour"), 0),
      enabled: fd.get("enabled") === "true",
      tags: csv(fd.get("tags")),
      notes: String(fd.get("notes") || "").trim(),
    };

    const authExpires = isoFromLocal(fd.get("auth_expires_at"));
    const rateReset = isoFromLocal(fd.get("rate_limit_reset_at"));
    const dailyReset = isoFromLocal(fd.get("daily_reset_at"));
    const monthlyReset = isoFromLocal(fd.get("monthly_reset_at"));
    if (authExpires) payload.auth_expires_at = authExpires;
    if (rateReset) payload.rate_limit_reset_at = rateReset;
    if (dailyReset) payload.daily_reset_at = dailyReset;
    if (monthlyReset) payload.monthly_reset_at = monthlyReset;

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
      if (!window.confirm(`Trigger 15m failover cooldown for ${provider}/${account}?`)) return;
      try {
        const out = await runAccountFailover(provider, account, 900);
        notify(`Failover applied to ${out.affected_profiles} profiles (${provider}/${account}).`);
        await refreshAll();
      } catch (err) {
        notify(err.message, false);
      }
      return;
    }

    const deleteRaw = e.target?.dataset?.deleteAccount;
    if (!deleteRaw) return;
    const [provider, account] = deleteRaw.split("::");
    if (!provider || !account) return;
    if (!window.confirm(`Delete account telemetry for ${provider}/${account}?`)) return;

    try {
      await api(`v2/accounts?provider=${encodeURIComponent(provider)}&account=${encodeURIComponent(account)}`, {
        method: "DELETE",
      });
      notify(`Deleted telemetry for ${provider}/${account}.`);
      await refreshSummary();
    } catch (err) {
      notify(err.message, false);
    }
  };

  $("accounts-grid")?.addEventListener("click", accountActionHandler);
  $("accounts-table-body")?.addEventListener("click", accountActionHandler);
  $("sla-alerts")?.addEventListener("click", accountActionHandler);

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
      priority: Number(fd.get("priority") || 0),
      enabled: fd.get("enabled") === "true",
      budget_daily_usd: Number(fd.get("budget_daily_usd") || 0),
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
      await api("v2/secrets", {
        method: "POST",
        body: JSON.stringify({ name: fd.get("name"), value: fd.get("value") }),
      });
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
        body: JSON.stringify({
          profile_id: fd.get("profile_id"),
          env_var: fd.get("env_var"),
          secret_key: fd.get("secret_key"),
        }),
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
      priority: Number(fd.get("priority") || 100),
      frontends: csv(fd.get("frontends")),
      task_classes: csv(fd.get("task_classes")),
      allow_providers: csv(fd.get("allow_providers")),
      deny_providers: csv(fd.get("deny_providers")),
      require_any_tag: csv(fd.get("require_any_tag")),
      require_auth_methods: csv(fd.get("require_auth_methods")),
      max_budget_daily_usd: Number(fd.get("max_budget_daily_usd") || 0),
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
      cooldown_seconds: Number(fd.get("cooldown_seconds") || 0),
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
}

async function init() {
  bindEvents();
  bindSectionObserver();
  setActiveSection("overview");
  try {
    await refreshAll();
    notify("Dashboard connected.");
  } catch (err) {
    notify(err.message, false);
  }
}

init();
