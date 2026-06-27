let apiBase = "http://localhost:18080";
let currentRoom = null;

const listMeta = {
  learning: {
    empty: "学习清单是空的",
    added: "已添加学习任务",
  },
  optimization: {
    empty: "任务清单是空的",
    added: "已添加任务",
  },
};

const elements = {
  apiStatus: document.querySelector("#apiStatus"),
  statusDot: document.querySelector("#statusDot"),
  roomName: document.querySelector("#roomName"),
  changeRoomButton: document.querySelector("#changeRoomButton"),
  totalCount: document.querySelector("#totalCount"),
  doneCount: document.querySelector("#doneCount"),
  progressRate: document.querySelector("#progressRate"),
  panels: [...document.querySelectorAll(".task-panel[data-kind]")],
};

let tasks = [];

async function request(path, options = {}) {
  const response = await fetch(`${apiBase}${path}`, {
    headers: { "Content-Type": "application/json", ...options.headers },
    ...options,
  });

  if (response.status === 204) {
    return null;
  }

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.message || "请求失败");
  }

  return data;
}

async function loadConfig() {
  try {
    const response = await fetch("/config.json");
    if (!response.ok) {
      throw new Error("config request failed");
    }

    const config = await response.json();
    apiBase = config.apiBase || apiBase;
  } catch (error) {
    console.warn("Using default API base:", apiBase);
  }
}

function loadCurrentRoom() {
  const params = new URLSearchParams(window.location.search);
  const roomID = Number(params.get("roomId"));
  const roomName = params.get("roomName");

  if (roomID > 0 && roomName) {
    currentRoom = {
      id: roomID,
      name: roomName,
    };
    localStorage.setItem("taskRoom", JSON.stringify(currentRoom));
    return;
  }

  try {
    const savedRoom = JSON.parse(localStorage.getItem("taskRoom"));
    if (savedRoom && Number(savedRoom.id) > 0 && savedRoom.name) {
      currentRoom = {
        id: Number(savedRoom.id),
        name: savedRoom.name,
      };
    }
  } catch (error) {
    localStorage.removeItem("taskRoom");
  }
}

function requireRoom() {
  if (currentRoom) {
    elements.roomName.textContent = currentRoom.name;
    return true;
  }

  window.location.href = "./home.html";
  return false;
}

async function checkHealth() {
  try {
    await request("/health");
    elements.apiStatus.textContent = "在线";
    elements.statusDot.className = "status-dot is-online";
  } catch (error) {
    elements.apiStatus.textContent = "未连接";
    elements.statusDot.className = "status-dot is-offline";
  }
}

async function loadTasks() {
  try {
    const data = await request(`/api/rooms/${currentRoom.id}/tasks`);
    tasks = data.items || [];
    renderAll();
    setAllMessages("列表已更新");
  } catch (error) {
    setAllMessages(error.message || "无法读取任务，请先启动后端服务");
  }
}

async function createTask(event) {
  event.preventDefault();

  const form = event.currentTarget;
  const kind = form.dataset.kind;
  const titleInput = form.elements.title;
  const levelInput = form.elements.level;
  const title = titleInput.value.trim();

  if (!title) {
    setMessage(kind, "任务标题不能为空");
    return;
  }

  try {
    const task = await request(`/api/rooms/${currentRoom.id}/tasks`, {
      method: "POST",
      body: JSON.stringify({
        title,
        level: levelInput.value,
        kind,
      }),
    });

    tasks = [task, ...tasks];
    form.reset();
    renderAll();
    setMessage(kind, listMeta[kind].added);
  } catch (error) {
    setMessage(kind, error.message);
  }
}

async function toggleTask(id, kind) {
  try {
    const updated = await request(`/api/rooms/${currentRoom.id}/tasks/${id}/toggle`, { method: "PATCH" });
    tasks = tasks.map((task) => (task.id === id ? updated : task));
    renderAll();
    setMessage(kind, updated.done ? "任务已完成" : "任务已恢复");
  } catch (error) {
    setMessage(kind, error.message);
  }
}

async function deleteTask(id, kind) {
  try {
    await request(`/api/rooms/${currentRoom.id}/tasks/${id}`, { method: "DELETE" });
    tasks = tasks.filter((task) => task.id !== id);
    renderAll();
    setMessage(kind, "任务已删除");
  } catch (error) {
    setMessage(kind, error.message);
  }
}

function renderAll() {
  updateMetrics();
  for (const panel of elements.panels) {
    renderPanel(panel.dataset.kind);
  }
}

function renderPanel(kind) {
  const panel = getPanel(kind);
  const list = panel.querySelector('[data-role="task-list"]');
  const items = tasks.filter((task) => normalizeKind(task.kind) === kind);

  if (items.length === 0) {
    list.innerHTML = `<li class="empty">${listMeta[kind].empty}</li>`;
    return;
  }

  list.innerHTML = items.map((task) => renderTask(task, kind)).join("");
}

function renderTask(task, kind) {
  const createdAt = new Date(task.createdAt).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });

  return `
    <li class="task-item ${task.done ? "is-done" : ""}">
      <button class="toggle-button" type="button" aria-label="切换任务状态" data-action="toggle" data-kind="${kind}" data-id="${task.id}">✓</button>
      <div>
        <p class="task-title">${escapeHTML(task.title)}</p>
        <p class="task-meta">${escapeHTML(task.level)} · ${createdAt}</p>
      </div>
      <button class="danger-button" type="button" aria-label="删除任务" data-action="delete" data-kind="${kind}" data-id="${task.id}">×</button>
    </li>
  `;
}

function updateMetrics() {
  const done = tasks.filter((task) => task.done).length;
  const progress = tasks.length === 0 ? 0 : Math.round((done / tasks.length) * 100);

  elements.totalCount.textContent = tasks.length;
  elements.doneCount.textContent = done;
  elements.progressRate.textContent = `${progress}%`;
}

function setMessage(kind, text) {
  const panel = getPanel(kind);
  panel.querySelector('[data-role="message"]').textContent = text;
}

function setAllMessages(text) {
  for (const panel of elements.panels) {
    setMessage(panel.dataset.kind, text);
  }
}

function getPanel(kind) {
  return elements.panels.find((panel) => panel.dataset.kind === kind);
}

function normalizeKind(kind) {
  return kind === "optimization" ? "optimization" : "learning";
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

for (const panel of elements.panels) {
  const kind = panel.dataset.kind;
  panel.querySelector("form").addEventListener("submit", createTask);
  panel.querySelector('[data-action="refresh"]').addEventListener("click", loadTasks);
  panel.querySelector('[data-role="task-list"]').addEventListener("click", (event) => {
    const button = event.target.closest("button[data-action]");
    if (!button) {
      return;
    }

    const id = Number(button.dataset.id);
    if (button.dataset.action === "toggle") {
      toggleTask(id, kind);
    }
    if (button.dataset.action === "delete") {
      deleteTask(id, kind);
    }
  });
}

elements.changeRoomButton.addEventListener("click", () => {
  localStorage.removeItem("taskRoom");
  window.location.href = "./home.html";
});

async function init() {
  await loadConfig();
  loadCurrentRoom();
  if (!requireRoom()) {
    return;
  }
  checkHealth();
  loadTasks();
}

init();
