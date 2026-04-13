const $ = (id) => document.getElementById(id);

async function api(path, options = {}) {
  const res = await fetch(path, {
    headers: { "content-type": "application/json", ...(options.headers || {}) },
    ...options,
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : {};
  if (!res.ok) {
    const msg = data?.error || `HTTP ${res.status}`;
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

function flash(id, msg, ok = true) {
  const el = $(id);
  el.className = `flash ${ok ? "ok" : "err"}`;
  el.textContent = msg;
  setTimeout(() => {
    if (el.textContent === msg) {
      el.textContent = "";
      el.className = "flash";
    }
  }, 5000);
}

function pretty(id, data) {
  $(id).textContent = JSON.stringify(data, null, 2);
}

function healthPill(profileEntry) {
  const h = profileEntry.health;
  if (!h) return `<span class="pill">no health</span>`;
  if (h.recent_error_rate_percent <= 2) {
    return `<span class="pill ok">${h.recent_error_rate_percent.toFixed(2)}% err</span>`;
  }
  return `<span class="pill bad">${h.recent_error_rate_percent.toFixed(2)}% err</span>`;
}

function leasePill(profileEntry) {
  if (!profileEntry.lease) return `<span class="pill">free</span>`;
  return `<span class="pill ok mono">${profileEntry.lease.owner}</span>`;
}

function renderStats(summary) {
  const counts = summary.counts || {};
  const stats = [
    ["Profiles", counts.profiles || 0],
    ["Accounts", counts.accounts || 0],
    ["Providers", counts.providers || 0],
    ["Policies", counts.policies || 0],
    ["Active Leases", counts.active_leases || 0],
  ];
  $("stats").innerHTML = stats
    .map(
      ([k, v]) => `<div class="stat"><div class="k">${k}</div><div class="v">${v}</div></div>`
    )
    .join("");
  $("as-of").textContent = `Updated: ${new Date(summary.time_utc).toLocaleString()}`;
}

function renderProfiles(summary) {
  const body = $("profiles-body");
  const rows = (summary.profiles || []).map((p) => {
    return `<tr>
      <td class="mono">${p.profile.id}</td>
      <td>${p.profile.provider}</td>
      <td>${p.profile.frontend}</td>
      <td>${p.profile.account || "default"}</td>
      <td>${p.profile.priority}</td>
      <td>${leasePill(p)}</td>
      <td>${healthPill(p)}</td>
      <td><button class="btn danger" data-delete-profile="${p.profile.id}">Delete</button></td>
    </tr>`;
  });
  body.innerHTML = rows.join("") || `<tr><td colspan="8">No profiles</td></tr>`;
}

async function refreshSummary() {
  const summary = await api("v2/dashboard/summary");
  renderStats(summary);
  renderProfiles(summary);
  pretty("policy-output", summary.policies || []);
}

async function refreshSecrets() {
  const secrets = await api("v2/secrets");
  pretty("secrets-output", secrets.items || []);
}

async function refreshAdapters() {
  const adapters = await api("v2/adapters");
  pretty("adapters-output", adapters);
}

async function refreshAll() {
  await Promise.all([refreshSummary(), refreshSecrets(), refreshAdapters()]);
}

function bindEvents() {
  $("refresh-all").addEventListener("click", () => refreshAll().catch((e) => alert(e.message)));
  $("refresh-adapters").addEventListener("click", () => refreshAdapters().catch((e) => alert(e.message)));

  $("profile-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
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
    };
    try {
      await api("v2/profiles", { method: "POST", body: JSON.stringify(payload) });
      flash("profile-flash", `Profile ${payload.id} saved`);
      e.target.reset();
      await refreshSummary();
    } catch (err) {
      flash("profile-flash", err.message, false);
    }
  });

  $("profiles-body").addEventListener("click", async (e) => {
    const id = e.target?.dataset?.deleteProfile;
    if (!id) return;
    if (!confirm(`Delete profile ${id}?`)) return;
    try {
      await api(`v2/profiles?id=${encodeURIComponent(id)}`, { method: "DELETE" });
      await refreshSummary();
    } catch (err) {
      alert(err.message);
    }
  });

  $("route-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const payload = {
      frontend: fd.get("frontend") || "",
      task_class: fd.get("task_class") || "coding",
      required_protocol: fd.get("required_protocol") || "",
      preferred_providers: csv(fd.get("preferred_providers")),
      require_tags: csv(fd.get("require_tags")),
      owner: fd.get("owner") || "dashboard",
    };
    try {
      const out = await api("v2/route/candidates", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      pretty("route-output", out);
    } catch (err) {
      pretty("route-output", { error: err.message });
    }
  });

  $("secret-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      await api("v2/secrets", {
        method: "POST",
        body: JSON.stringify({ name: fd.get("name"), value: fd.get("value") }),
      });
      flash("secret-flash", `Secret ${fd.get("name")} stored`);
      e.target.reset();
      await refreshSecrets();
    } catch (err) {
      flash("secret-flash", err.message, false);
    }
  });

  $("binding-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      await api("v2/secret-bindings", {
        method: "POST",
        body: JSON.stringify({
          profile_id: fd.get("profile_id"),
          env_var: fd.get("env_var"),
          secret_key: fd.get("secret_key"),
        }),
      });
      flash("secret-flash", "Binding saved");
      e.target.reset();
      await refreshSummary();
    } catch (err) {
      flash("secret-flash", err.message, false);
    }
  });

  $("policy-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
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
    try {
      await api("v2/policies", { method: "POST", body: JSON.stringify(payload) });
      flash("policy-flash", `Policy ${payload.name} saved`);
      e.target.reset();
      await refreshSummary();
    } catch (err) {
      flash("policy-flash", err.message, false);
    }
  });
}

async function init() {
  bindEvents();
  try {
    await refreshAll();
  } catch (err) {
    alert(err.message);
  }
}

init();
