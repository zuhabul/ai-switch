const $ = (id) => document.getElementById(id);

let authToken = localStorage.getItem("aiswitch_api_token") || "";
let toastTimer = null;
let latestSummary = null;

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
  stats.innerHTML = cards
    .map(([k, v]) => `<div class="stat"><div class="k">${k}</div><div class="v">${v}</div></div>`)
    .join("");
  const updated = summary.time_utc ? new Date(summary.time_utc) : new Date();
  $("as-of").textContent = updated.toLocaleString();

  const providers = Object.entries(summary.providers || {}).sort((a, b) => a[0].localeCompare(b[0]));
  const providerStrip = $("providers-strip");
  providerStrip.innerHTML = providers
    .map(([name, count]) => `<span class="provider-pill">${name}<strong>${count}</strong></span>`)
    .join("");
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
              <div class="bar ${usageBarClass(dailyUsage).replace("bar ", "")}"><i style="width:${Math.min(100, dailyUsage)}%"></i></div>
              <div class="subhead">${accountUsageText(account.daily_used_usd, account.daily_limit_usd, account.daily_remaining_usd)}</div>
            </div>

            <div class="usage">
              <div class="k">Monthly Usage ${monthlyUsage.toFixed(1)}%</div>
              <div class="bar ${usageBarClass(monthlyUsage).replace("bar ", "")}"><i style="width:${Math.min(100, monthlyUsage)}%"></i></div>
              <div class="subhead">${accountUsageText(account.monthly_used_usd, account.monthly_limit_usd, account.monthly_remaining_usd)}</div>
            </div>

            <div class="account-actions">
              <button class="btn btn-muted" type="button" data-load-account="${accountKey(account.provider, account.account)}">Edit</button>
              <button class="btn btn-danger" type="button" data-delete-account="${account.provider}::${account.account}">Delete</button>
            </div>
          </article>`;
      })
      .join("");
  }

  const tableBody = $("accounts-table-body");
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
          <button class="btn btn-muted" type="button" data-load-account="${accountKey(account.provider, account.account)}">Edit</button>
          <button class="btn btn-danger" type="button" data-delete-account="${account.provider}::${account.account}">Delete</button>
        </td>
      </tr>`;
    })
    .join("");

  if (!tableBody.innerHTML.trim()) {
    tableBody.innerHTML = `<tr><td colspan="10">No account telemetry available.</td></tr>`;
  }
}

function renderProfiles(summary) {
  const body = $("profiles-body");
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
      <td><button class="btn btn-danger" type="button" data-delete-profile="${profile.id}">Delete</button></td>
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

  tokenInput.value = authToken;

  saveTokenBtn.addEventListener("click", async () => {
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

  $("refresh-all").addEventListener("click", async (e) => {
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

  $("refresh-adapters").addEventListener("click", async (e) => {
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

  $("account-filter-provider").addEventListener("change", () => {
    if (latestSummary) renderAccounts(latestSummary);
  });
  $("account-filter-status").addEventListener("change", () => {
    if (latestSummary) renderAccounts(latestSummary);
  });

  $("account-form").addEventListener("submit", async (e) => {
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

  $("accounts-grid").addEventListener("click", accountActionHandler);
  $("accounts-table-body").addEventListener("click", accountActionHandler);

  $("profile-form").addEventListener("submit", async (e) => {
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

  $("profiles-body").addEventListener("click", async (e) => {
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

  $("route-form").addEventListener("submit", async (e) => {
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

  $("secret-form").addEventListener("submit", async (e) => {
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

  $("binding-form").addEventListener("submit", async (e) => {
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

  $("policy-form").addEventListener("submit", async (e) => {
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

  $("incident-form").addEventListener("submit", async (e) => {
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
