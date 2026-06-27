let apiBase = "http://localhost:18080";

const historyKey = "taskRoomHistory";
const form = document.querySelector("#roomForm");
const input = document.querySelector("#roomName");
const message = document.querySelector("#roomMessage");
const historyList = document.querySelector("#historyList");
const clearHistoryButton = document.querySelector("#clearHistoryButton");

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

async function request(path, options = {}) {
  const response = await fetch(`${apiBase}${path}`, {
    headers: { "Content-Type": "application/json", ...options.headers },
    ...options,
  });

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.message || "请求失败");
  }

  return data;
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  const name = input.value.trim();
  if (!name) {
    message.textContent = "房间名称不能为空";
    input.focus();
    return;
  }

  const button = form.querySelector(".enter-button");
  button.disabled = true;
  message.textContent = "正在进入房间...";

  try {
    const room = await request("/api/rooms", {
      method: "POST",
      body: JSON.stringify({ name }),
    });

    rememberRoom(room);
    enterRoom(room);
  } catch (error) {
    message.textContent = error.message;
    button.disabled = false;
  }
});

historyList.addEventListener("click", (event) => {
  const button = event.target.closest("button[data-room-id]");
  if (!button) {
    return;
  }

  const roomID = Number(button.dataset.roomId);
  const room = readHistory().find((item) => Number(item.id) === roomID);
  if (!room) {
    renderHistory();
    return;
  }

  rememberRoom(room);
  enterRoom(room);
});

clearHistoryButton.addEventListener("click", () => {
  localStorage.removeItem(historyKey);
  renderHistory();
});

function enterRoom(room) {
  localStorage.setItem("taskRoom", JSON.stringify(room));
  window.location.href = `./tasks.html?roomId=${encodeURIComponent(room.id)}&roomName=${encodeURIComponent(room.name)}`;
}

function rememberRoom(room) {
  const history = readHistory();
  const nextHistory = [
    {
      id: Number(room.id),
      name: room.name,
      visitedAt: new Date().toISOString(),
    },
    ...history.filter((item) => Number(item.id) !== Number(room.id)),
  ].slice(0, 6);

  localStorage.setItem(historyKey, JSON.stringify(nextHistory));
  renderHistory();
}

function readHistory() {
  try {
    const history = JSON.parse(localStorage.getItem(historyKey));
    return Array.isArray(history) ? history.filter((item) => item && item.id && item.name) : [];
  } catch (error) {
    localStorage.removeItem(historyKey);
    return [];
  }
}

function renderHistory() {
  const history = readHistory();
  clearHistoryButton.disabled = history.length === 0;

  if (history.length === 0) {
    historyList.innerHTML = `<p class="history-empty">还没有进入过房间</p>`;
    return;
  }

  historyList.innerHTML = history
    .map((room) => {
      const visitedAt = room.visitedAt ? formatVisitedAt(room.visitedAt) : "最近";
      return `
        <button class="history-item" type="button" data-room-id="${room.id}">
          ${escapeHTML(room.name)}
          <span>${visitedAt}</span>
        </button>
      `;
    })
    .join("");
}

function formatVisitedAt(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "最近";
  }

  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
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

loadConfig();
renderHistory();
