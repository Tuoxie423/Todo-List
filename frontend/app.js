let apiBase = "http://localhost:18080";
let currentUser = null;
let currentList = null;
const authKey = "taskUser";
const urgencyClassMap = {
  今天: "today",
  本周: "week",
  近期: "soon",
  有空: "later",
};
const urgencyRankMap = {
  今天: 0,
  本周: 1,
  近期: 2,
  有空: 3,
};

const elements = {
  apiStatus: document.querySelector("#apiStatus"),
  statusDot: document.querySelector("#statusDot"),
  roomName: document.querySelector("#roomName"),
  changeListButton: document.querySelector("#changeListButton"),
  totalCount: document.querySelector("#totalCount"),
  doneCount: document.querySelector("#doneCount"),
  progressRate: document.querySelector("#progressRate"),
  panel: document.querySelector(".task-panel"),
  taskForm: document.querySelector(".task-form"),
  taskList: document.querySelector('[data-role="task-list"]'),
  message: document.querySelector('[data-role="message"]'),
};

let tasks = [];

async function request(path, options = {}) {
  const headers = { "Content-Type": "application/json", ...options.headers };
  if (currentUser && currentUser.token) {
    headers.Authorization = `Bearer ${currentUser.token}`;
  }

  const response = await fetch(`${apiBase}${path}`, {
    headers,
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

function loadCurrentUser() {
  try {
    const savedUser = JSON.parse(localStorage.getItem(authKey));
    if (savedUser && Number(savedUser.id) > 0 && savedUser.username && savedUser.token) {
      currentUser = {
        id: Number(savedUser.id),
        username: savedUser.username,
        token: savedUser.token,
      };
    }
  } catch (error) {
    localStorage.removeItem(authKey);
  }
}

function requireUser() {
  if (currentUser) {
    return true;
  }

  localStorage.removeItem("taskList");
  window.location.href = "./home.html";
  return false;
}
function loadCurrentList() {
  const params = new URLSearchParams(window.location.search);
  const roomID = Number(params.get("roomId"));
  const roomName = params.get("roomName");

  if (roomID > 0 && roomName) {
    currentList = {
      id: roomID,
      name: roomName,
    };
    localStorage.setItem("taskList", JSON.stringify(currentList));
    return;
  }

  try {
    const savedList = JSON.parse(localStorage.getItem("taskList"));
    if (savedList && Number(savedList.id) > 0 && savedList.name) {
      currentList = {
        id: Number(savedList.id),
        name: savedList.name,
      };
    }
  } catch (error) {
    localStorage.removeItem("taskList");
  }
}

function requireList() {
  if (currentList) {
    elements.roomName.textContent = currentList.name;
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
    const data = await request(`/api/rooms/${currentList.id}/tasks`);
    tasks = sortTasks(data.items || []);
    renderAll();
    setMessage("列表已更新");
  } catch (error) {
    setMessage(error.message || "无法读取任务，请先启动后端服务");
  }
}

async function createTask(event) {
  event.preventDefault();

  const form = event.currentTarget;
  const titleInput = form.elements.title;
  const levelInput = form.elements.level;
  const title = titleInput.value.trim();

  if (!title) {
    setMessage("任务标题不能为空");
    return;
  }

  try {
    const task = await request(`/api/rooms/${currentList.id}/tasks`, {
      method: "POST",
      body: JSON.stringify({
        title,
        level: levelInput.value,
      }),
    });

    tasks = sortTasks([task, ...tasks]);
    form.reset();
    renderAll();
    setMessage("已添加任务");
  } catch (error) {
    setMessage(error.message);
  }
}

async function toggleTask(id) {
  try {
    const updated = await request(`/api/rooms/${currentList.id}/tasks/${id}/toggle`, { method: "PATCH" });
    tasks = tasks.map((task) => (task.id === id ? updated : task));
    renderAll();
    setMessage(updated.done ? "任务已完成" : "任务已恢复");
  } catch (error) {
    setMessage(error.message);
  }
}

async function deleteTask(id) {
  try {
    await request(`/api/rooms/${currentList.id}/tasks/${id}`, { method: "DELETE" });
    tasks = tasks.filter((task) => task.id !== id);
    renderAll();
    setMessage("任务已删除");
  } catch (error) {
    setMessage(error.message);
  }
}

function renderAll() {
  updateMetrics();
  renderTaskList();
}

function renderTaskList() {
  if (tasks.length === 0) {
    elements.taskList.innerHTML = `<li class="empty">任务清单是空的</li>`;
    return;
  }

  elements.taskList.innerHTML = tasks.map(renderTask).join("");
}

function renderTask(task) {
  const createdAt = new Date(task.createdAt).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });

  const urgency = normalizeUrgency(task.level);
  const urgencyClass = urgencyClassMap[urgency];

  return `
    <li class="task-item ${task.done ? "is-done" : ""}">
      <button class="toggle-button" type="button" aria-label="切换任务状态" data-action="toggle" data-id="${task.id}">✓</button>
      <div>
        <p class="task-title">${escapeHTML(task.title)}</p>
        <p class="task-meta"><span class="urgency-badge urgency-${urgencyClass}">${escapeHTML(urgency)}</span><span>${createdAt}</span></p>
      </div>
      <button class="danger-button" type="button" aria-label="删除任务" data-action="delete" data-id="${task.id}">×</button>
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

function setMessage(text) {
  elements.message.textContent = text;
}

function normalizeUrgency(level) {
  return Object.prototype.hasOwnProperty.call(urgencyClassMap, level) ? level : "有空";
}

function sortTasks(items) {
  return [...items].sort((left, right) => {
    const leftRank = urgencyRankMap[normalizeUrgency(left.level)];
    const rightRank = urgencyRankMap[normalizeUrgency(right.level)];
    if (leftRank !== rightRank) {
      return leftRank - rightRank;
    }

    const leftTime = new Date(left.createdAt).getTime() || 0;
    const rightTime = new Date(right.createdAt).getTime() || 0;
    return rightTime - leftTime;
  });
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

elements.taskForm.addEventListener("submit", createTask);
elements.panel.querySelector('[data-action="refresh"]').addEventListener("click", loadTasks);
elements.taskList.addEventListener("click", (event) => {
  const button = event.target.closest("button[data-action]");
  if (!button) {
    return;
  }

  const id = Number(button.dataset.id);
  if (button.dataset.action === "toggle") {
    toggleTask(id);
  }
  if (button.dataset.action === "delete") {
    deleteTask(id);
  }
});

elements.changeListButton.addEventListener("click", () => {
  localStorage.removeItem("taskList");
  window.location.href = "./home.html";
});

async function init() {
  await loadConfig();
  loadCurrentUser();
  if (!requireUser()) {
    return;
  }
  loadCurrentList();
  if (!requireList()) {
    return;
  }
  checkHealth();
  loadTasks();
}

init();
