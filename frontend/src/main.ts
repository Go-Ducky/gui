import "./style.css";
import { Events } from "@wailsio/runtime";
import * as Service from "../bindings/github.com/go-ducky/gui/internal/guiservice/service.js";
import type { Info, MsgView, SessionView } from "../bindings/github.com/go-ducky/gui/internal/guiservice/models.js";

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

const state = {
  provider: "" as string,
  model: "" as string,
  workDir: "" as string,
  name: "" as string,
  onboarded: true as boolean,
  running: false as boolean,
  messages: [] as MsgView[],
  sessions: [] as SessionView[],
  streamEl: null as HTMLElement | null,
  streamInfo: null as HTMLDivElement | null,
  streamText: "" as string,
  currentTools: [] as { el: HTMLElement; name: string }[],
  approvalId: "" as string,
  showingTypes: new Set<HTMLElement>(),
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const $ = (sel: string) => document.querySelector<HTMLElement>(sel)!;

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// Light rendering: fenced ``` code blocks become <pre>, everything else is
// escaped text preserved with newlines.
function renderBody(text: string): string {
  const parts = text.split(/```/);
  let out = "";
  for (let i = 0; i < parts.length; i++) {
    if (i % 2 === 1) {
      const code = parts[i].replace(/^[a-zA-Z0-9_+-]+\n?/, "");
      out += `<pre class="code">${esc(code)}</pre>`;
    } else {
      out += esc(parts[i]);
    }
  }
  return out;
}

function toast(msg: string, ms = 3800) {
  const el = document.createElement("div");
  el.className = "toast";
  el.textContent = msg;
  $("#toast-root").appendChild(el);
  setTimeout(() => el.remove(), ms);
}

function setStatus(msg: string) {
  $("#statusLine").textContent = msg;
}

function openModal(html: string): () => void {
  const root = $("#modal-root");
  root.hidden = false;
  root.innerHTML = `<div class="modal">${html}</div>`;
  const close = () => {
    root.hidden = true;
    root.innerHTML = "";
  };
  root.addEventListener(
    "click",
    (e) => {
      if (e.target === root) close();
    },
    { once: true },
  );
  return close;
}

// ---------------------------------------------------------------------------
// Thread rendering
// ---------------------------------------------------------------------------

function addUserMsg(text: string) {
  const wrap = document.createElement("div");
  wrap.className = "msg user";
  wrap.innerHTML = `<div class="msg-role">You</div><div class="msg-body">${esc(text)}</div>`;
  $("#thread").appendChild(wrap);
  scrollThread();
}

function ensureAssistant(): { body: HTMLElement; tools: HTMLElement } {
  if (state.streamEl) {
    const body = state.streamEl.querySelector<HTMLElement>(".msg-body")!;
    const tools = state.streamEl.querySelector<HTMLElement>(".tools")!;
    return { body, tools };
  }
  const wrap = document.createElement("div");
  wrap.className = "msg assistant";
  wrap.innerHTML = `
    <div class="msg-role">GoDucky</div>
    <div class="msg-body caret"></div>
    <div class="tools"></div>`;
  $("#thread").appendChild(wrap);
  state.streamEl = wrap;
  state.streamText = "";
  scrollThread();
  return {
    body: wrap.querySelector<HTMLElement>(".msg-body")!,
    tools: wrap.querySelector<HTMLElement>(".tools")!,
  };
}

function renderTool(name: string, args: string, isError = false, content = "") {
  const { tools } = ensureAssistant();
  const item = document.createElement("details") as HTMLDetailsElement;
  item.className = "tool";
  item.open = !isError;
  item.innerHTML = `
    <summary class="tool-head">
      <span class="tool-icon">${isError ? "✗" : "⚙"}</span>
      <span class="tool-name">${esc(name)}</span>
      <span class="tool-state">${isError ? "error" : "running"}</span>
    </summary>
    <div class="tool-detail">${esc(args)}</div>`;
  tools.appendChild(item);
  state.currentTools.push({ el: item, name });
  if (isError) {
    item.classList.add("tool-error");
    const stateSpan = item.querySelector<HTMLElement>(".tool-state")!;
    stateSpan.textContent = "error";
    const detail = item.querySelector<HTMLElement>(".tool-detail")!;
    detail.textContent = content || args;
    const icon = item.querySelector<HTMLElement>(".tool-icon")!;
    icon.textContent = "✗";
    item.classList.remove("tool-open");
  }
  scrollThread();
}

function finishTool(name: string, content: string, isError: boolean) {
  const stack = state.currentTools;
  const idx = [...stack].reverse().findIndex((t) => t.name === name);
  if (idx < 0) return;
  const item = stack[stack.length - 1 - idx].el;
  const stateSpan = item.querySelector<HTMLElement>(".tool-state")!;
  const icon = item.querySelector<HTMLElement>(".tool-icon")!;
  stateSpan.textContent = isError ? "error" : "done";
  stateSpan.style.color = isError ? "var(--red)" : "var(--green)";
  icon.textContent = isError ? "✗" : "✓";
  if (isError) {
    item.classList.add("tool-error");
    (item as HTMLDetailsElement).open = true;
    item.querySelector<HTMLElement>(".tool-detail")!.textContent = content || "...";
  } else {
    const detail = item.querySelector<HTMLElement>(".tool-detail")!;
    detail.textContent = content;
    item.dataset.content = content;
  }
  stack.splice(stack.length - 1 - idx, 1);
}

function scrollThread() {
  const t = $("#thread");
  t.scrollTop = t.scrollHeight;
}

// ---------------------------------------------------------------------------
// Sidebar / sessions
// ---------------------------------------------------------------------------

function renderSessions() {
  const list = $("#sessionsList");
  list.innerHTML = "";
  if (!state.sessions.length) {
    list.innerHTML = `<div class="dim" style="font-size:12px;padding:6px 10px;">No saved chats yet.</div>`;
    return;
  }
  for (const s of state.sessions) {
    const active = s.name === state.name;
    const item = document.createElement("div");
    item.className = `session-item${active ? " active" : ""}`;
    const when = s.updated || "";
    item.innerHTML = `
      <span class="session-title">${esc(s.name)}</span>
      <span class="session-preview">${esc(s.preview || "")}</span>
      <span class="session-meta"><span>${esc(s.provider || "")}</span><span>${esc(when)}</span></span>
      <div class="session-actions">
        <button data-act="rename" title="Rename">✎</button>
        <button data-act="delete" title="Delete">🗑</button>
      </div>`;
    item.addEventListener("click", () => {
      if (!state.running) void resumeChat(s.name);
    });
    item.querySelector<HTMLButtonElement>('[data-act="rename"]')!.addEventListener("click", async (e) => {
      e.stopPropagation();
      const name = prompt("New session name:", s.name);
      if (name && name.trim()) {
        await Service.RenameSession(s.name, name.trim());
        await refresh();
      }
    });
    item.querySelector<HTMLButtonElement>('[data-act="delete"]')!.addEventListener("click", async (e) => {
      e.stopPropagation();
      await Service.DeleteSession(s.name);
      if (state.name === s.name) state.name = "";
      await refresh();
    });
    list.appendChild(item);
  }
}

async function refresh() {
  const info = await Service.GetInfo();
  if (info) applyInfo(info);
  renderSessions();
}

function applyInfo(i: Info) {
  state.provider = i.provider;
  state.model = i.model;
  state.workDir = i.work_dir;
  state.name = i.name;
  state.onboarded = i.onboarded;
  state.messages = i.messages ?? [];
  $("#providerBtn").textContent = i.provider || "provider";
  $("#modelBtn").textContent = i.model || "model";
  $("#workDirPill").innerHTML = `📁 ${esc(i.work_dir || "")}`;
}

async function resumeChat(name: string) {
  await Service.Resume(name);
  await refresh();
  renderInfoMessages();
}

function renderInfoMessages() {
  $("#thread").innerHTML = "";
  if (state.messages.length === 0) {
    $("#welcome").hidden = false;
    return;
  }
  $("#welcome").hidden = true;
  for (const m of state.messages) {
    if (m.role === "user" || m.role === "system") {
      const wrap = document.createElement("div");
      wrap.className = "msg user";
      wrap.innerHTML = `<div class="msg-role">${m.role === "system" ? "System" : "You"}</div><div class="msg-body">${renderBody(m.text)}</div>`;
      $("#thread").appendChild(wrap);
    } else {
      const wrap = document.createElement("div");
      wrap.className = "msg assistant";
      wrap.innerHTML = `<div class="msg-role">GoDucky</div><div class="msg-body">${renderBody(m.text)}</div>`;
      $("#thread").appendChild(wrap);
    }
  }
  scrollThread();
}

// ---------------------------------------------------------------------------
// Send / stop
// ---------------------------------------------------------------------------

async function send() {
  const input = $("#input") as HTMLTextAreaElement;
  const text = input.value.trim();
  if (!text || state.running) return;
  input.value = "";
  input.style.height = "auto";
  setRunning(true);
  addUserMsg(text);
  ensureAssistant();
  renderTool("agent", "Starting…");
  await Service.Send(text);
}

async function stop() {
  setRunning(false);
  await Service.Stop();
  if (state.streamEl) {
    finalizeAssistant(" (stopped)");
  }
}

function setRunning(r: boolean) {
  state.running = r;
  (($("#sendBtn") as HTMLButtonElement).disabled = r);
  $("#stopBtn").hidden = !r;
  $("#newChatBtn").hidden = r;
  renderSessions();
}

function finalizeAssistant(suffix = "") {
  if (!state.streamEl) return;
  state.streamText += suffix;
  const body = state.streamEl.querySelector<HTMLElement>(".msg-body")!;
  body.className = "msg-body";
  body.innerHTML = renderBody(state.streamText);
  state.streamEl = null;
  state.streamText = "";
}

// ---------------------------------------------------------------------------
// Pickers & modals
// ---------------------------------------------------------------------------

const CLOUD_PROVIDERS = ["groq", "openai", "anthropic", "gemini", "openrouter"];

async function showProviderPicker() {
  const providers = (await Service.Providers()) ?? [];
  openModal(`
    <h2>Provider</h2>
    <div class="list">
      ${providers
        .map(
          (p) => `
        <button class="list-item ${p === state.provider ? "" : ""}" data-p="${esc(p)}">
          ${esc(p)}
          ${p === state.provider ? " <span style='color:var(--accent)'>● current</span>" : ""}
        </button>`,
        )
        .join("")}
    </div>
    <div class="modal-buttons"><button class="btn" data-close>Close</button></div>`);
  $("#modal-root").querySelector(".modal")!.addEventListener("click", async (e) => {
    const b = (e.target as HTMLElement).closest<HTMLButtonElement>("[data-p]");
    if (b) {
      await switchProvider(b.dataset.p!);
      $("#modal-root").click();
    } else if ((e.target as HTMLElement).closest("[data-close]")) {
      $("#modal-root").click();
    }
  });
}

async function switchProvider(p: string) {
  if (state.running) {
    toast("Stop the run before switching providers.");
    return;
  }
  try {
    await Service.SwitchProvider(p);
  } catch (err) {
    toast(String(err));
  }
  await refresh();
  // Cloud providers need a key
  if (CLOUD_PROVIDERS.includes(p)) {
    const hasKey = await Service.HasAPIKey(p);
    if (!hasKey) showKeyModal(p);
  }
}

async function showModelPicker() {
  openModal(`
    <h2>Model</h2>
    <div class="list" id="modelList"><div class="dim">Loading…</div></div>
    <div class="modal-buttons"><button class="btn" data-close>Close</button></div>`);
  const mod = $("#modal-root").querySelector(".modal")!;
  mod.addEventListener("click", async (e) => {
    if ((e.target as HTMLElement).closest("[data-close]")) {
      $("#modal-root").click();
      return;
    }
    const b = (e.target as HTMLElement).closest<HTMLButtonElement>("[data-m]");
    if (b) {
      try {
        await Service.SetModel(b.dataset.m!);
        await refresh();
        $("#modal-root").click();
      } catch (err) {
        toast(String(err));
      }
    }
  });
  const list = mod.querySelector<HTMLElement>("#modelList")!;
  let models: string[] = [];
  try {
    models = (await Service.GetModels(state.provider)) ?? [];
  } catch {
    models = [];
  }
  if (!models.length) {
    list.innerHTML = `<div class="dim">No models listed for ${esc(state.provider)}. Switch provider in Settings.</div>`;
    return;
  }
  list.innerHTML = models
    .map(
      (m) => `
    <button class="list-item" data-m="${esc(m)}">
      ${esc(m)}
      ${m === state.model ? " <span style='color:var(--accent)'>● current</span>" : ""}
      <span class="sub">${esc(state.provider)}</span>
    </button>`,
    )
    .join("");
}

function showKeyModal(provider: string) {
  openModal(`
    <h2>API key — ${esc(provider)}</h2>
    <p>Paste a key for <b>${esc(provider)}</b>. It is stored in your config.<br/>
    <span class="dim mono">Env var override: ${esc(providerConfigEnv(provider))}</span></p>
    <input class="input-field" id="keyInput" type="password" placeholder="sk-…" autocomplete="off"/>
    <div class="modal-buttons">
      <button class="btn" data-cancel>Cancel</button>
      <button class="btn primary" id="keySave">Save & continue</button>
    </div>`);
  const input = $("#keyInput") as HTMLInputElement;
  input.focus();
  const doSave = async () => {
    const key = input.value.trim();
    if (!key) return;
    try {
      await Service.SaveAPIKey(provider, key);
      await refresh();
      $("#modal-root").click();
      toast(`Key for ${provider} saved.`);
    } catch (err) {
      toast(String(err));
    }
  };
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") void doSave();
  });
  $("#keySave").addEventListener("click", doSave);
  $("#modal-root").querySelector('[data-cancel]')!.addEventListener("click", () => $("#modal-root").click());
}

function providerConfigEnv(p: string): string {
  switch (p) {
    case "groq":
      return "GROQ_API_KEY";
    case "openai":
      return "OPENAI_API_KEY";
    case "anthropic":
      return "ANTHROPIC_API_KEY";
    case "gemini":
      return "GEMINI_API_KEY";
    case "openrouter":
      return "OPENROUTER_API_KEY";
    default:
      return "";
  }
}

function showWorkDirModal() {
  openModal(`
    <h2>Working directory</h2>
    <p>GoDucky reads, writes, edits and runs commands in this folder.</p>
    <input class="input-field" id="wdInput" value="${esc(state.workDir)}"/>
    <div class="modal-buttons"><button class="btn" data-cancel>Cancel</button><button class="btn primary" data-ok>Save</button></div>`);
  const input = $("#wdInput") as HTMLInputElement;
  input.focus();
  input.select();
  const doSet = async () => {
    const dir = input.value.trim();
    if (!dir) return;
    try {
      await Service.SetWorkDir(dir);
      await refresh();
      $("#modal-root").click();
      setStatus(`Working directory: ${dir}`);
    } catch (err) {
      toast(String(err));
    }
  };
  $("#modal-root").querySelector('[data-ok]')!.addEventListener("click", doSet);
  $("#modal-root").querySelector('[data-cancel]')!.addEventListener("click", () => $("#modal-root").click());
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") void doSet();
  });
}

async function showSettingsModal() {
  const hasKey = await Service.HasAPIKey(state.provider);
  openModal(`
    <h2>Settings</h2>
    <div class="list">
      <button class="list-item" data-act="key"><span>API key — ${esc(state.provider)}</span><span class="sub">${hasKey ? "Set ✓" : "Not set"}</span></button>
      <button class="list-item" data-act="dir"><span>Working directory</span><span class="sub mono">${esc(state.workDir)}</span></button>
      <button class="list-item" data-act="provider"><span>Switch provider</span><span class="sub">current: ${esc(state.provider)}</span></button>
      <button class="list-item" data-act="model"><span>Change model</span><span class="sub">current: ${esc(state.model)}</span></button>
      <button class="list-item" data-act="save"><span>Save this chat</span><span class="sub">${esc(state.name || "untitled")}</span></button>
    </div>
    <div class="field-row">
      <div><span class="field-label">Auto-approve actions</span>
      <span class="field-hint">Run file edits & commands without asking</span></div>
      <label class="switch"><input type="checkbox" id="autoApprove"/> <span class="slider"></span></label>
    </div>
    <div class="modal-buttons"><button class="btn" data-close>Close</button></div>`);
  const mod = $("#modal-root").querySelector(".modal")!;
  const check = mod.querySelector<HTMLInputElement>("#autoApprove")!;
  try {
    const info = await Service.GetInfo();
    check.checked = !!(info && (info as any).auto_approve);
  } catch {}
  check.addEventListener("change", async (e) => {
    await Service.SetAutoApprove((e.target as HTMLInputElement).checked).catch((err) => toast(String(err)));
  });
  mod.addEventListener("click", async (e) => {
    if ((e.target as HTMLElement).closest("[data-close]")) {
      $("#modal-root").click();
      return;
    }
    const b = (e.target as HTMLElement).closest<HTMLButtonElement>("[data-act]");
    if (!b) return;
    const act = b.dataset.act!;
    if (act === "key") {
      $("#modal-root").click();
      showKeyModal(state.provider);
    } else if (act === "dir") {
      $("#modal-root").click();
      showWorkDirModal();
    } else if (act === "provider") {
      $("#modal-root").click();
      await showProviderPicker();
    } else if (act === "model") {
      $("#modal-root").click();
      await showModelPicker();
    } else if (act === "save") {
      try {
        await Service.SaveSession("");
        await refresh();
        toast("Chat saved.");
      } catch (err) {
        toast(String(err));
      }
    }
  });
}

// ---------------------------------------------------------------------------
// Persistence
// ---------------------------------------------------------------------------

async function newChat() {
  if (state.running) {
    toast("Stop the run first.");
    return;
  }
  await Service.NewChat();
  await refresh();
  $("#thread").innerHTML = "";
  $("#welcome").hidden = false;
  renderSessions();
}

// ---------------------------------------------------------------------------
// Onboarding
// ---------------------------------------------------------------------------

async function checkOnboarding() {
  const info = await Service.GetInfo();
  if (!info || info.onboarded) return;
  renderOnboarding();
}

function renderOnboarding() {
  openModal(`
    <h2>Welcome to GoDucky 👋</h2>
    <p>An AI coding agent that runs in this folder. Two ways to get going:</p>
    <div class="list">
      <button class="list-item" data-flow="local"><span>🦆 Run locally with Ollama</span><span class="sub">Free, private, runs on your machine</span></button>
      <button class="list-item" data-flow="cloud"><span>☁️ Use a cloud provider</span><span class="sub">Groq, OpenAI, Anthropic, Gemini, OpenRouter</span></button>
    </div>
    <div class="modal-buttons"><button class="btn" data-skip>Skip for now</button></div>`);
  $("#modal-root").querySelector(".modal")!.addEventListener("click", (e) => {
    const t = e.target as HTMLElement;
    const flow = t.closest<HTMLButtonElement>("[data-flow]")?.dataset.flow;
    if (flow === "local") {
      $("#modal-root").click();
      void onboardLocal();
    } else if (flow === "cloud") {
      $("#modal-root").click();
      openModal(`
        <h2>Cloud provider</h2>
        <p>Which provider do you want to use?</p>
        <div class="list" id="cloudList">${CLOUD_PROVIDERS.map((p) => `<button class="list-item" data-p="${p}">${p}</button>`).join("")}</div>`);
      $("#modal-root").querySelector("#cloudList")!.addEventListener("click", (ev) => {
        const b = (ev.target as HTMLElement).closest<HTMLButtonElement>("[data-p]");
        if (!b) return;
        $("#modal-root").click();
        showKeyModal(b.dataset.p!);
        finishOnboarding();
      });
    } else if (t.closest("[data-skip]")) {
      $("#modal-root").click();
      void Service.ApplySetup().catch((err) => toast(String(err)));
    }
  });
}

async function finishOnboarding() {
  try {
    await Service.ApplySetup();
    state.onboarded = true;
  } catch (err) {
    toast(String(err));
  }
}

async function onboardLocal() {
  try {
    const st = (await Service.OllamaStatus()) as any;
    if (!st?.installed) {
      openModal(`
        <h2>Install Ollama</h2>
        <p>GoDucky will download and install Ollama for you now.</p>
        <div class="progress-bar"><div></div></div>
        <div class="dim" id="ollamaProgress">Starting install…</div>`);
      await Service.InstallOllama();
      toast("Ollama install started in background.");
      pollOllamaInstall();
      return;
    }
    if (!st?.running) {
      toast("Ollama installed but not running — start it, then pick a model.");
    }
    void pickLocalModel(true);
  } catch (err) {
    toast(String(err));
  }
}

async function pollOllamaInstall() {
  const bar = document.querySelector<HTMLElement>("#ollamaProgress");
  const fill = document.querySelector<HTMLElement>(".progress-bar > div");
  const ticks = ["Starting…", "Downloading…", "Installing…", "Almost there…", "Checking…"];
  if (!bar) return;
  for (let i = 0; i < ticks.length; i++) {
    bar.textContent = ticks[i];
    if (fill) fill.style.width = `${18 + i * 20}%`;
    await sleep(2500);
  }
  const st = (await Service.OllamaStatus()) as any;
  if (fill) fill.style.width = "100%";
  bar.textContent = st?.installed ? "Ollama installed and running ✓" : "Install not detected — you may need to start Ollama manually.";
  if (st?.installed) {
    setTimeout(() => {
      if (!document.getElementById("modelList")) void pickLocalModel(true);
    }, 400);
  }
}
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

async function pickLocalModel(force = false) {
  const recommended = (await Service.RecommendedLocalModels()) ?? [];
  openModal(`
    <h2>Pick a local model</h2>
    <p>Ollama will download the model on first use (a few GB).</p>
    <div class="list" id="modelList">
      ${recommended.map((m) => `<button class="list-item" data-m="${esc(m)}">${esc(m)}<span class="sub">Recommended</span></button>`).join("")}
    </div>
    <div class="modal-buttons"><button class="btn" data-skip>Not now</button></div>`);
  force;
  $("#modal-root").querySelector("#modelList")!.addEventListener("click", async (e) => {
    const b = (e.target as HTMLElement).closest<HTMLButtonElement>("[data-m]");
    if (!b) return;
    const model = b.dataset.m!;
    const has = await Service.HasModel(model);
    $("#modal-root").click();
    if (!has) {
      toast(`Pulling ${model}…`);
      Service.StartLocalModelPulls([model]);
    }
    try {
      await Service.SwitchProvider("ollama");
      await Service.SetModel(model);
    } catch (err) {
      toast(String(err));
    }
    await finishOnboarding();
    await refresh();
  });
  $("#modal-root").querySelector('[data-skip]')!.addEventListener("click", () => {
    $("#modal-root").click();
    void Service.SwitchProvider("ollama").then(async () => {
      await finishOnboarding();
      await refresh();
    });
  });
}

// ---------------------------------------------------------------------------
// Events
// ---------------------------------------------------------------------------

function wire() {
  Events.On("agent:stream", (e: any) => {
    const { body } = ensureAssistant();
    if (body.classList.contains("caret")) body.classList.remove("caret");
    state.streamText += e.data?.text ?? "";
    body.textContent = state.streamText;
    scrollThread();
  });

  Events.On("agent:status", (e: any) => {
    setStatus(e.data?.msg ?? "");
  });

  Events.On("agent:tool:start", (e: any) => {
    renderTool(e.data?.name ?? "tool", e.data?.args ?? "");
  });

  Events.On("agent:tool:end", (e: any) => {
    finishTool(e.data?.name ?? "", e.data?.content ?? "", !!e.data?.is_error);
  });

  Events.On("agent:complete", (e: any) => {
    setRunning(false);
    const stopped = !!e.data?.stopped;
    finalizeAssistant(stopped ? "\n\n(stopped)" : "");
    setStatus(stopped ? "Run stopped." : "");
    void refresh();
  });

  Events.On("agent:error", (e: any) => {
    setRunning(false);
    setStatus(String(e.data?.error ?? "error"));
    toast(String(e.data?.error ?? "error"));
    if (state.streamEl) finalizeAssistant("\n\n[error]");
    void refresh();
  });

  Events.On("agent:approval", (e: any) => {
    const d = e.data ?? {};
    state.approvalId = String(d.id ?? "");
    openModal(`
      <h2>Approve action</h2>
      <p>${esc(d.desc || "GoDucky wants to make a change.")}</p>
      <pre class="tool-detail">${esc(typeof d.args === "string" ? d.args : JSON.stringify(d.args ?? "", null, 2))}</pre>
      <div class="modal-buttons">
        <button class="btn danger" id="deny">Deny</button>
        <button class="btn primary" id="allow">Allow</button>
      </div>`);
    $("#allow").addEventListener("click", () => {
      $("#modal-root").click();
      void Service.Approve(state.approvalId, true);
    });
    $("#deny").addEventListener("click", () => {
      $("#modal-root").click();
      void Service.Approve(state.approvalId, false);
    });
  });

  Events.On("ollama:op", (e: any) => {
    const d = e.data ?? {};
    if (d.error) {
      toast(`Ollama ${d.action} ${d.model}: ${d.error}`);
    } else {
      toast(`Ollama ${d.action} ${d.model} done.`);
      Service.GetModels(state.provider).then((models) => {
        if (d.model && models?.includes(d.model) && !state.onboarded) {
          void finishOnboarding();
        }
      });
    }
  });
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

async function init() {
  wire();

  try {
    const v = await Service.AppVersion();
    $("#versionTag").textContent = `v${v}`;
  } catch {}

  $("#input").addEventListener("input", (e) => {
    const el = e.target as HTMLTextAreaElement;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 180) + "px";
  });
  $("#input").addEventListener("keydown", (e) => {
    const k = e as KeyboardEvent;
    if (k.key === "Enter" && !k.shiftKey) {
      k.preventDefault();
      void send();
    }
  });
  $("#sendBtn").addEventListener("click", () => void send());
  $("#stopBtn").addEventListener("click", () => void stop());
  $("#newChatBtn").addEventListener("click", () => void newChat());
  $("#providerBtn").addEventListener("click", () => void showProviderPicker());
  $("#modelBtn").addEventListener("click", () => void showModelPicker());
  $("#workDirPill").addEventListener("click", showWorkDirModal);
  $("#settingsBtn").addEventListener("click", () => void showSettingsModal());
  $("#saveBtn").addEventListener("click", async () => {
    try {
      await Service.SaveSession("");
      await refresh();
      setStatus("Chat saved.");
    } catch (err) {
      toast(String(err));
    }
  });

  await refresh();
  renderInfoMessages();
  await checkOnboarding();
}

init().catch((err) => {
  console.error(err);
  setStatus("Failed to start UI: " + err);
});