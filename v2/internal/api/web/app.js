const $ = (id) => document.getElementById(id);
let authToken = localStorage.getItem("aiswitch_api_token") || "";
let toastTimer = null;

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

function flash(id, msg, ok = true) {
  const el = $(id);
  if (!el) return;
  el.className = `flash ${ok ? "ok" : "err"}`;
  el.textContent = msg;
  setTimeout(() => {
    if (el.textContent === msg) {
      el.textContent = "";
      el.className = "flash";
    }
  }, 4500);
}

function pretty(id, data) {
  const el = $(id);
  if (!el) return;
  el.textContent = JSON.stringify(data, null, 2);
}

function healthPill(entry) {
  const health = entry.health;
  if (!health) return `<span class="pill">no health</span>`;
  const errPct = Number(health.recent_error_rate_percent || 0);
  if (errPct <= 2) {
    return `<span class="pill ok">${errPct.toFixed(2)}% err</span>`;
  }
  return `<span class="pill bad">${errPct.toFixed(2)}% err</span>`;
}

function leasePill(entry) {
  if (!entry.lease) return `<span class="pill">free</span>`;
  return `<span class="pill ok mono">${entry.lease.owner || "assigned"}</span>`;
}

function renderStats(summary) {
  const counts = summary.counts || {};
  const cards = [
    ["Profiles", counts.profiles || 0],
    ["Accounts", counts.accounts || 0],
    ["Providers", counts.providers || 0],
    ["Policies", counts.policies || 0],
    ["Active Leases", counts.active_leases || 0],
    ["Incidents", counts.incidents || 0],
  ];

  const target = $("stats");
  if (target) {
    target.innerHTML = cards
      .map(([label, value]) => `<div class="stat"><div class="k">${label}</div><div class="v">${value}</div></div>`)
      .join("");
  }

  const updated = summary.time_utc ? new Date(summary.time_utc) : new Date();
  $("as-of").textContent = `Synced ${updated.toLocaleString()}`;
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
      <td><button class="btn btn-danger" type="button" data-delete-profile="${profile.id}">Delete</button></td>
    </tr>`;
  });

  body.innerHTML = rows.join("") || `<tr><td colspan="8">No profiles configured yet.</td></tr>`;
}

async function refreshSummary() {
  const summary = await api("v2/dashboard/summary");
  renderStats(summary);
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
  const links = document.querySelectorAll(".section-nav a");
  for (const link of links) {
    const active = link.getAttribute("href") === `#${sectionId}`;
    link.classList.toggle("is-active", active);
  }
}

function bindSectionObserver() {
  const sections = document.querySelectorAll("main section[id], article[id]");
  if (!sections.length || !("IntersectionObserver" in window)) return;

  const observer = new IntersectionObserver(
    (entries) => {
      const visible = entries
        .filter((entry) => entry.isIntersecting)
        .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];
      if (visible?.target?.id) {
        setActiveSection(visible.target.id);
      }
    },
    { threshold: [0.25, 0.45, 0.7], rootMargin: "-15% 0px -50% 0px" }
  );

  sections.forEach((section) => observer.observe(section));

  document.querySelectorAll('.section-nav a[href^="#"]').forEach((link) => {
    link.addEventListener("click", () => {
      const id = link.getAttribute("href")?.slice(1);
      if (id) setActiveSection(id);
    });
  });
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
    setButtonLoading(button, true, "Refresh");
    try {
      await refreshAll();
      notify("Dashboard refreshed.");
    } catch (err) {
      notify(err.message, false);
    } finally {
      setButtonLoading(button, false, "Refresh");
    }
  });

  $("refresh-adapters").addEventListener("click", async (e) => {
    const button = e.currentTarget;
    setButtonLoading(button, true, "Adapters");
    try {
      await refreshAdapters();
      notify("Adapter contract updated.");
    } catch (err) {
      notify(err.message, false);
    } finally {
      setButtonLoading(button, false, "Adapters");
    }
  });

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
      notify(`Profile ${payload.id} created.`);
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
      await refreshSummary();
      notify(`Profile ${id} deleted.`);
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
      const out = await api("v2/route/candidates", {
        method: "POST",
        body: JSON.stringify(payload),
      });
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
