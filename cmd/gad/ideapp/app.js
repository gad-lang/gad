"use strict";
// Bundled Gad IDE frontend. Talks to the /api/ide/* backend exposed by
// `gad ide`. Vanilla JS, no build step — a richer React/CodeMirror UI can be
// served instead via `gad ide --static <dist>`.

const $ = (id) => document.getElementById(id);
const api = {
  async json(method, url, body) {
    const r = await fetch(url, {
      method,
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    });
    const data = await r.json().catch(() => ({}));
    if (!r.ok) throw new Error(data.error || r.statusText);
    return data;
  },
  workspace: () => api.json("GET", "api/ide/workspace"),
  tree: () => api.json("GET", "api/ide/tree"),
  read: (p) => api.json("GET", "api/ide/file?path=" + encodeURIComponent(p)),
  write: (p, content) => api.json("PUT", "api/ide/file", { path: p, content }),
  mkfile: (p) => api.json("PUT", "api/ide/file", { path: p, content: "" }),
  del: (p) => api.json("POST", "api/ide/delete", { path: p }),
  config: () => api.json("GET", "api/ide/config"),
  saveConfig: (doc) => api.json("PUT", "api/ide/config", doc),
  modules: () => api.json("GET", "api/ide/modules"),
  format: (source) => api.json("POST", "api/ide/format", { source }),
  run: (req) => api.json("POST", "api/ide/run", req),
  dbgStart: (req) => api.json("POST", "api/ide/debug/start", req),
  dbgCmd: (session, command) => api.json("POST", "api/ide/debug/command", { session, command }),
};

// --- state ------------------------------------------------------------------
const state = {
  open: [],        // [{path, content, saved, runCfg}]
  active: -1,
  config: {},
  modules: [],
  debug: null,     // {session, path}
};

// --- theme ------------------------------------------------------------------
function curTheme() { return document.documentElement.dataset.theme === "dark" ? "dark" : "light"; }
$("themeBtn").onclick = () => {
  const t = curTheme() === "dark" ? "light" : "dark";
  document.documentElement.dataset.theme = t;
  localStorage.setItem("gad-theme", t);
  saveLayout();
};

// --- file tree --------------------------------------------------------------
async function refreshTree() {
  const root = await api.tree();
  const el = $("tree");
  el.innerHTML = "";
  (root.children || []).forEach((c) => el.appendChild(renderNode(c)));
}
function renderNode(node) {
  const wrap = document.createElement("div");
  const row = document.createElement("div");
  row.className = "node";
  row.dataset.path = node.path;
  row.textContent = (node.dir ? "📁 " : "📄 ") + node.name;
  wrap.appendChild(row);
  if (node.dir) {
    const kids = document.createElement("div");
    kids.className = "children";
    (node.children || []).forEach((c) => kids.appendChild(renderNode(c)));
    let collapsed = false;
    row.onclick = () => { collapsed = !collapsed; kids.style.display = collapsed ? "none" : ""; };
    wrap.appendChild(kids);
  } else {
    row.onclick = () => openFile(node.path);
  }
  return wrap;
}
function markActiveInTree() {
  document.querySelectorAll(".tree .node").forEach((n) => {
    n.classList.toggle("active", state.active >= 0 && n.dataset.path === state.open[state.active].path);
  });
}

// --- tabs & editor ----------------------------------------------------------
async function openFile(path) {
  const i = state.open.findIndex((f) => f.path === path);
  if (i >= 0) { setActive(i); return; }
  const data = await api.read(path);
  state.open.push({ path, content: data.content, saved: true, runCfg: defaultRunCfg(path) });
  setActive(state.open.length - 1);
  renderTabs();
}
function setActive(i) { state.active = i; renderTabs(); renderEditor(); markActiveInTree(); }
function closeTab(i) {
  state.open.splice(i, 1);
  if (state.active >= state.open.length) state.active = state.open.length - 1;
  renderTabs(); renderEditor(); markActiveInTree();
}
function renderTabs() {
  const el = $("tabs"); el.innerHTML = "";
  state.open.forEach((f, i) => {
    const t = document.createElement("div");
    t.className = "tab" + (i === state.active ? " active" : "") + (f.saved ? "" : " dirty");
    const name = document.createElement("span"); name.className = "name";
    name.textContent = f.path.split("/").pop();
    t.appendChild(name);
    const x = document.createElement("span"); x.className = "x"; x.textContent = "✕";
    x.onclick = (e) => { e.stopPropagation(); closeTab(i); };
    t.appendChild(x);
    t.onclick = () => setActive(i);
    el.appendChild(t);
  });
}
function renderEditor() {
  const wrap = $("editorWrap");
  wrap.innerHTML = "";
  if (state.active < 0) { wrap.innerHTML = '<div class="empty">Open a file from the explorer</div>'; return; }
  const f = state.open[state.active];
  const ta = document.createElement("textarea");
  ta.spellcheck = false;
  ta.value = f.content;
  ta.oninput = () => { f.content = ta.value; if (f.saved) { f.saved = false; renderTabs(); } };
  ta.onkeydown = (e) => {
    if (e.key === "Tab") { e.preventDefault(); insertAtCursor(ta, "\t"); f.content = ta.value; }
    if ((e.ctrlKey || e.metaKey) && e.key === "s") { e.preventDefault(); saveActive(); }
  };
  wrap.appendChild(ta);
  wrap._ta = ta;
}
function insertAtCursor(ta, text) {
  const s = ta.selectionStart, e = ta.selectionEnd;
  ta.value = ta.value.slice(0, s) + text + ta.value.slice(e);
  ta.selectionStart = ta.selectionEnd = s + text.length;
}
function activeFile() { return state.active >= 0 ? state.open[state.active] : null; }

// --- actions ----------------------------------------------------------------
async function saveActive() {
  const f = activeFile(); if (!f) return;
  await api.write(f.path, f.content);
  f.saved = true; renderTabs(); status("saved " + f.path);
}
async function formatActive() {
  const f = activeFile(); if (!f) return;
  const res = await api.format(f.content);
  if (res.ok) { f.content = res.source; f.saved = false; renderEditor(); renderTabs(); status("formatted"); }
  else showDiagnostics(res.diagnostics);
}
$("saveBtn").onclick = saveActive;
$("fmtBtn").onclick = formatActive;
$("newBtn").onclick = async () => {
  const name = prompt("New file path (relative to workspace):", "untitled.gad");
  if (!name) return;
  await api.mkfile(name); await refreshTree(); openFile(name);
};

// --- run --------------------------------------------------------------------
function defaultRunCfg(path) {
  const saved = (state.config.ide && state.config.ide.run && state.config.ide.run[path]) || {};
  return { args: saved.args || [], disabled: saved.disabled || [], safe: !!saved.safe, saveOut: saved.saveOut || "" };
}
$("runBtn").onclick = () => openRunDialog(false);
$("dbgBtn").onclick = () => openRunDialog(true);

function openRunDialog(debug) {
  const f = activeFile(); if (!f) { status("open a file first"); return; }
  const cfg = f.runCfg;
  const bg = document.createElement("div"); bg.className = "modal-bg";
  const mods = state.modules.map((m) =>
    `<label class="ck"><input type="checkbox" data-mod="${m.name}" ${cfg.disabled.includes(m.name) ? "" : "checked"}> ${m.name}${m.unsafe ? " <span class='muted'>(unsafe)</span>" : ""}</label>`
  ).join("");
  bg.innerHTML = `<div class="modal">
    <h3>${debug ? "Debug" : "Run"} ${f.path}</h3>
    <div class="row"><label>Arguments (one per line)</label><textarea id="m_args" rows="3">${cfg.args.join("\n")}</textarea></div>
    ${debug ? `<div class="row"><label>Breakpoints (lines, comma-separated)</label><input id="m_bp" value=""></div>
      <div class="row"><label class="ck"><input type="checkbox" id="m_entry"> Stop on entry</label></div>` : ""}
    <div class="row"><label>Builtin modules (checked = enabled)</label><div class="mods">${mods}</div></div>
    <div class="row"><label class="ck"><input type="checkbox" id="m_safe" ${cfg.safe ? "checked" : ""}> Safe mode (disable all unsafe modules)</label></div>
    <div class="row"><label>Save stdout+stderr to file (optional)</label><input id="m_out" value="${cfg.saveOut}" placeholder="output.log"></div>
    <div class="actions"><button id="m_cancel">Cancel</button><button id="m_go">${debug ? "Start Debug" : "Run"}</button></div>
  </div>`;
  document.body.appendChild(bg);
  const close = () => bg.remove();
  bg.onclick = (e) => { if (e.target === bg) close(); };
  $("m_cancel").onclick = close;
  $("m_go").onclick = async () => {
    cfg.args = $("m_args").value.split("\n").map((s) => s.trim()).filter(Boolean);
    cfg.disabled = state.modules.filter((m) => !bg.querySelector(`[data-mod="${m.name}"]`).checked).map((m) => m.name);
    cfg.safe = $("m_safe").checked;
    cfg.saveOut = $("m_out").value.trim();
    persistRunCfg(f.path, cfg);
    if (debug) {
      const bp = ($("m_bp").value || "").split(",").map((s) => parseInt(s.trim(), 10)).filter((n) => n > 0);
      close(); startDebug(f, bp, $("m_entry").checked);
    } else {
      close(); await doRun(f, cfg);
    }
  };
}
async function doRun(f, cfg) {
  status("running…");
  if (!f.saved) await api.write(f.path, f.content).then(() => { f.saved = true; renderTabs(); });
  try {
    const res = await api.run({ path: f.path, source: f.content, args: cfg.args, disabled: cfg.disabled, safe: cfg.safe, saveOut: cfg.saveOut });
    showRun(res); status(res.ok ? "done" : "error");
  } catch (e) { showText($("outPane"), String(e), "diag"); status("error"); }
  selectPane("out");
}
function showRun(res) {
  let s = "";
  if (res.stdout) s += res.stdout;
  if (res.stderr) s += res.stderr;
  if (res.ok && res.result) s += "\n⇦ " + res.result + "\n";
  if (res.diagnostics) res.diagnostics.forEach((d) => { s += `${d.line}:${d.column} ${d.message}\n`; });
  showText($("outPane"), s || "(no output)", res.ok ? "" : "diag");
}

// --- debug ------------------------------------------------------------------
async function startDebug(f, breakpoints, stopOnEntry) {
  if (!f.saved) await api.write(f.path, f.content).then(() => { f.saved = true; renderTabs(); });
  status("debugging…");
  try {
    const cfg = f.runCfg;
    const res = await api.dbgStart({
      source: f.content, breakpoints, stopOnEntry,
      path: f.path, args: cfg.args, disabled: cfg.disabled, safe: cfg.safe,
    });
    state.debug = { session: res.session, path: f.path };
    $("dbgbar").style.display = res.state === "stopped" ? "" : "none";
    applyDebugResponse(res);
  } catch (e) { showText($("outPane"), String(e), "diag"); status("error"); }
}
document.querySelectorAll("#dbgbar button").forEach((b) => {
  b.onclick = async () => {
    if (!state.debug) return;
    if (b.dataset.cmd === "stop") { endDebug(); return; }
    const res = await api.dbgCmd(state.debug.session, b.dataset.cmd);
    applyDebugResponse(res);
  };
});
function applyDebugResponse(res) {
  if (res.output) appendText($("outPane"), res.output);
  if (res.state === "stopped") {
    status(`stopped (${res.reason}) at line ${res.line}`);
    renderStack(res.frames || []);
    renderLocals(res.locals || []);
    selectPane("stack");
  } else if (res.state === "terminated") {
    if (res.result) appendText($("outPane"), "\n⇦ " + res.result + "\n");
    if (res.error) appendText($("outPane"), "\n" + res.error + "\n");
    status("program exited"); endDebug();
  } else if (res.state === "error") {
    showText($("outPane"), res.error || "debug error", "diag");
    if (res.diagnostics) showDiagnostics(res.diagnostics);
    endDebug();
  }
}
function endDebug() { state.debug = null; $("dbgbar").style.display = "none"; }
let frameClickTimer = null;
function renderStack(frames) {
  const pane = $("stackPane");
  pane.innerHTML = "";
  if (!frames.length) { pane.textContent = "(empty)"; return; }
  frames.forEach((f) => {
    const file = f.file ? f.file.split("/").pop() + ":" : "";
    const div = document.createElement("div");
    div.className = "frame";
    div.style.cursor = "pointer";
    div.title = (f.file || "") + ":" + f.line + ":" + f.column + " — click to inspect, double-click to open";
    div.innerHTML = `<b>${escapeHtml(f.name || "main")}</b> <span class="muted">${escapeHtml(file)}${f.line}:${f.column}</span>`;
    // Single click shows this frame's locals; double click navigates.
    div.onclick = () => {
      if (frameClickTimer !== null) {
        clearTimeout(frameClickTimer); frameClickTimer = null;
        gotoFrame(f.file, f.line, f.column);
        return;
      }
      frameClickTimer = setTimeout(() => {
        frameClickTimer = null;
        pane.querySelectorAll(".frame").forEach((n) => n.classList.remove("selected"));
        div.classList.add("selected");
        renderLocals(f.locals || []);
        selectPane("locals");
      }, 250);
    };
    pane.appendChild(div);
  });
}
// gotoFrame opens the frame's file (if needed) and moves the cursor there.
async function gotoFrame(file, line, column) {
  try { await openFile(file); } catch (e) { return; }
  const ta = $("editorWrap")._ta;
  if (!ta) return;
  const lines = ta.value.split("\n");
  let pos = 0;
  for (let i = 0; i < line - 1 && i < lines.length; i++) pos += lines[i].length + 1;
  pos += Math.max(0, column - 1);
  ta.focus();
  ta.setSelectionRange(pos, pos);
  // Scroll the caret roughly into view.
  const approxLineH = ta.scrollHeight / Math.max(lines.length, 1);
  ta.scrollTop = Math.max(0, (line - 3) * approxLineH);
}
function renderLocals(locals) {
  if (!locals.length) { $("localsPane").textContent = "(no locals)"; return; }
  $("localsPane").innerHTML = '<table class="locals">' + locals.map((v) =>
    `<tr><td>${escapeHtml(v.name)}</td><td class="muted">${escapeHtml(v.type)}</td><td>${escapeHtml(v.value)}</td></tr>`).join("") + "</table>";
}

// --- output panes -----------------------------------------------------------
$("outTabs").querySelectorAll("button").forEach((b) => b.onclick = () => selectPane(b.dataset.pane));
function selectPane(pane) {
  $("outTabs").querySelectorAll("button").forEach((b) => b.classList.toggle("active", b.dataset.pane === pane));
  $("outPane").style.display = pane === "out" ? "" : "none";
  $("stackPane").style.display = pane === "stack" ? "" : "none";
  $("localsPane").style.display = pane === "locals" ? "" : "none";
}
$("clearOut").onclick = () => { $("outPane").textContent = ""; };
function showText(el, text, cls) { el.className = cls || ""; el.textContent = text; }
function appendText(el, text) { el.textContent += text; }
function showDiagnostics(diags) {
  showText($("outPane"), (diags || []).map((d) => `${d.line}:${d.column} ${d.message}`).join("\n"), "diag");
  selectPane("out");
}
function status(s) { $("status").textContent = s; }
function escapeHtml(s) { return String(s).replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" }[c])); }

// --- settings / config ------------------------------------------------------
$("cfgBtn").onclick = openConfigDialog;
async function openConfigDialog() {
  const doc = await api.config();
  state.config = doc || {};
  const fmt = state.config.fmt || {};
  const bg = document.createElement("div"); bg.className = "modal-bg";
  // These map to `gad fmt` keys `no-<k>-in-new-line`. Checked = expanded layout
  // (the default, no key); unchecking writes `no-…-in-new-line: true` to compact.
  const newlineFlags = [
    ["no-array-item-in-new-line", "Array items on new lines"],
    ["no-dict-item-in-new-line", "Dict items on new lines"],
    ["no-call-params-in-new-line", "Call params on new lines"],
  ];
  const flag = ([k, label]) => `<label class="ck"><input type="checkbox" data-noflag="${k}" ${fmt[k] === true ? "" : "checked"}> ${label}</label>`;
  bg.innerHTML = `<div class="modal">
    <h3>Settings</h3>
    <div class="row"><label>Formatter (.gad.yaml → fmt)</label>
      ${newlineFlags.map(flag).join("")}
      <label class="ck"><input type="checkbox" data-fmt="backup" ${fmt.backup ? "checked" : ""}> Keep .backup on format</label>
    </div>
    <div class="row"><label>Raw .gad.yaml</label><textarea id="cfgRaw" rows="8" style="font-family:ui-monospace,monospace">${escapeHtml(toYamlish(state.config))}</textarea></div>
    <div class="actions"><button id="c_cancel">Cancel</button><button id="c_save">Save</button></div>
  </div>`;
  document.body.appendChild(bg);
  const close = () => bg.remove();
  bg.onclick = (e) => { if (e.target === bg) close(); };
  $("c_cancel").onclick = close;
  $("c_save").onclick = async () => {
    const fmtObj = state.config.fmt || {};
    // Inverted `no-…` flags: checked (expanded, default) removes the key.
    bg.querySelectorAll("[data-noflag]").forEach((cb) => {
      const k = cb.dataset.noflag;
      if (cb.checked) delete fmtObj[k]; else fmtObj[k] = true;
    });
    bg.querySelectorAll("[data-fmt]").forEach((cb) => {
      const k = cb.dataset.fmt;
      if (cb.checked) fmtObj[k] = true; else delete fmtObj[k];
    });
    state.config.fmt = fmtObj;
    await api.saveConfig(state.config);
    status("settings saved"); close();
  };
}
// toYamlish renders a shallow object as readable YAML-ish text (display only).
function toYamlish(obj) { try { return JSON.stringify(obj, null, 2); } catch (e) { return "{}"; } }

// --- layout persistence (.gad.yaml ide key) ---------------------------------
let layoutTimer = null;
function saveLayout() {
  clearTimeout(layoutTimer);
  layoutTimer = setTimeout(async () => {
    state.config.ide = Object.assign({}, state.config.ide, {
      theme: curTheme(),
      sidebarWidth: $("sidebar").offsetWidth,
      outputHeight: $("output").offsetHeight,
    });
    try { await api.saveConfig(state.config); } catch (e) {}
  }, 500);
}
function persistRunCfg(path, cfg) {
  const ide = state.config.ide || (state.config.ide = {});
  const run = ide.run || (ide.run = {});
  run[path] = cfg;
  saveLayout();
}
function applyLayout() {
  const ide = state.config.ide || {};
  if (ide.theme) { document.documentElement.dataset.theme = ide.theme; }
  if (ide.sidebarWidth) $("sidebar").style.width = ide.sidebarWidth + "px";
  if (ide.outputHeight) $("output").style.height = ide.outputHeight + "px";
}

// --- resizers ---------------------------------------------------------------
function dragResize(gutter, target, axis) {
  gutter.addEventListener("mousedown", (e) => {
    e.preventDefault();
    const startPos = axis === "x" ? e.clientX : e.clientY;
    const startSize = axis === "x" ? target.offsetWidth : target.offsetHeight;
    const sign = axis === "y" ? -1 : 1; // output grows upward
    const move = (ev) => {
      const cur = axis === "x" ? ev.clientX : ev.clientY;
      const size = startSize + sign * (cur - startPos);
      target.style[axis === "x" ? "width" : "height"] = Math.max(60, size) + "px";
    };
    const up = () => { document.removeEventListener("mousemove", move); document.removeEventListener("mouseup", up); saveLayout(); };
    document.addEventListener("mousemove", move);
    document.addEventListener("mouseup", up);
  });
}
dragResize($("gutterX"), $("sidebar"), "x");
dragResize($("gutterY"), $("output"), "y");

// --- boot -------------------------------------------------------------------
(async function boot() {
  try {
    const ws = await api.workspace();
    $("ws").textContent = ws.root;
    state.config = await api.config();
    state.modules = await api.modules();
    applyLayout();
    await refreshTree();
    if (ws.openFile) openFile(ws.openFile);
    selectPane("out");
  } catch (e) { status("failed to start: " + e); }
})();
