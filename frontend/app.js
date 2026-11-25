const { api, storage, levels, dom } = window.Todo;

let currentUser = null;
let currentList = null;

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

function loadCurrentUser() {
  currentUser = storage.getCurrentUser();
}

function requireUser() {
  if (currentUser) {
    return true;
  }

  storage.clearCurrentRoom();
  window.location.href = "./home.html";
  return false;
}

function loadCurrentList() {
  const params = new URLSearchParams(window.location.search);
  const roomID = Number(params.get("roomId"));
  const roomName = params.get("roomName");

  if (roomID > 0 && roomName) {
    currentList = storage.saveCurrentRoom({ id: roomID, name: roomName });
    return;
  }

  currentList = storage.getCurrentRoom();
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
    await api.request("/health");
    elements.apiStatus.textContent = "在线";
    elements.statusDot.className = "status-dot is-online";
  } catch (error) {
    elements.apiStatus.textContent = "未连接";
    elements.statusDot.className = "status-dot is-offline";
  }
}

async function loadTasks() {
  try {
    const data = await api.request(`/api/rooms/${currentList.id}/tasks`);
    tasks = levels.sortTasks(data.items || []);
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
    const task = await api.request(`/api/rooms/${currentList.id}/tasks`, {
      method: "POST",
      body: JSON.stringify({
        title,
        level: levelInput.value,
      }),
    });

    tasks = levels.sortTasks([task, ...tasks]);
    form.reset();
    renderAll();
    setMessage("已添加任务");
  } catch (error) {
    setMessage(error.message);
  }
}

async function toggleTask(id) {
  try {
    const updated = await api.request(`/api/rooms/${currentList.id}/tasks/${id}/toggle`, { method: "PATCH" });
    tasks = tasks.map((task) => (task.id === id ? updated : task));
    renderAll();
    setMessage(updated.done ? "任务已完成" : "任务已恢复");
  } catch (error) {
    setMessage(error.message);
  }
}

async function deleteTask(id) {
  try {
    await api.request(`/api/rooms/${currentList.id}/tasks/${id}`, { method: "DELETE" });
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

  const urgency = levels.normalize(task.level);
  const urgencyClass = levels.classFor(urgency);

  return `
    <li class="task-item ${task.done ? "is-done" : ""}">
      <button class="toggle-button" type="button" aria-label="切换任务状态" data-action="toggle" data-id="${task.id}">✓</button>
      <div>
        <p class="task-title">${dom.escapeHTML(task.title)}</p>
        <p class="task-meta"><span class="urgency-badge urgency-${urgencyClass}">${dom.escapeHTML(urgency)}</span><span>${createdAt}</span></p>
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

function populateLevels() {
  elements.taskForm.elements.level.innerHTML = levels.list
    .map((level) => `<option>${level}</option>`)
    .join("");
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
  storage.clearCurrentRoom();
  window.location.href = "./home.html";
});

async function init() {
  populateLevels();
  await api.loadConfig();
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
