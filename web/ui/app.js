/* CYBEROS Web UI foundation — thin client over /api/v1. No business logic. */
async function getJSON(path) {
  const res = await fetch(path);
  if (!res.ok) throw new Error(path + " → " + res.status);
  return res.json();
}

function el(id) { return document.getElementById(id); }

function table(headers, rows) {
  if (!rows.length) return "<p class='muted'>None</p>";
  const th = headers.map(h => `<th>${h}</th>`).join("");
  const tr = rows.map(r => `<tr>${r.map(c => `<td>${c}</td>`).join("")}</tr>`).join("");
  return `<table><thead><tr>${th}</tr></thead><tbody>${tr}</tbody></table>`;
}

async function load() {
  try {
    const dash = await getJSON("/api/v1/dashboard");
    el("stat-projects").textContent = dash.projects ?? 0;
    el("stat-scopes").textContent = dash.scopes ?? 0;
    el("stat-tasks").textContent = dash.tasks ?? 0;
    el("stat-caps").textContent = dash.capabilities ?? 0;

    const projects = (await getJSON("/api/v1/projects")).projects || [];
    el("projects").innerHTML = table(
      ["ID", "Name", "Description"],
      projects.map(p => [p.id, p.name, p.description || ""])
    );

    const scopes = (await getJSON("/api/v1/scopes")).scopes || [];
    el("scopes").innerHTML = table(
      ["ID", "Project", "Name", "Status"],
      scopes.map(s => [s.id, s.project_id, s.name, `<span class="badge">${s.status}</span>`])
    );

    const tasks = (await getJSON("/api/v1/tasks")).tasks || [];
    el("tasks").innerHTML = table(
      ["ID", "Title", "State", "Scope", "Capability"],
      tasks.map(t => [
        t.id, t.title, `<span class="badge">${t.state}</span>`, t.scope_id, t.capability_id
      ])
    );

    const caps = (await getJSON("/api/v1/capabilities")).capabilities || [];
    el("capabilities").innerHTML = table(
      ["ID", "Name", "Type"],
      caps.map(c => [c.id, c.name || c.id, c.type || ""])
    );
  } catch (e) {
    console.error(e);
    el("projects").innerHTML = `<p class="error">${e.message}</p>`;
  }
}

load();
