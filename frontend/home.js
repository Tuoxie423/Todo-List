let apiBase = "http://localhost:18080";
let currentUser = null;

const authKey = "taskUser";
const accountHistoryKey = "taskAccountHistory";
const roomKey = "taskList";
const authForm = document.querySelector("#authForm");
const usernameInput = document.querySelector("#username");
const passwordInput = document.querySelector("#password");
const authMessage = document.querySelector("#authMessage");
const accountHistoryList = document.querySelector("#accountHistoryList");
const clearAccountHistoryButton = document.querySelector("#clearAccountHistoryButton");
const signedPanel = document.querySelector("#signedPanel");
const currentUsername = document.querySelector("#currentUsername");
const logoutButton = document.querySelector("#logoutButton");
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
  const headers = { "Content-Type": "application/json", ...options.headers };
  if (currentUser && currentUser.token) {
    headers.Authorization = `Bearer ${currentUser.token}`;
  }

  const response = await fetch(`${apiBase}${path}`, {
    headers,
    ...options,
  });

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.message || "Request failed");
  }

  return data;
}

authForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  const username = usernameInput.value.trim();
  const password = passwordInput.value;
  if (!username) {
    authMessage.textContent = "\u7528\u6237\u540d\u4e0d\u80fd\u4e3a\u7a7a";
    usernameInput.focus();
    return;
  }
  if (password.length < 6) {
    authMessage.textContent = "\u5bc6\u7801\u81f3\u5c11\u9700\u8981 6 \u4f4d";
    passwordInput.focus();
    return;
  }

  const button = authForm.querySelector(".enter-button");
  button.disabled = true;
  authMessage.textContent = "\u6b63\u5728\u767b\u5f55...";

  try {
    const data = await request("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });

    rememberAccount(data.user);
    setCurrentUser(data.user);
    passwordInput.value = "";
    authMessage.textContent = data.registered ? "\u65b0\u7528\u6237\u5df2\u81ea\u52a8\u6ce8\u518c" : "\u767b\u5f55\u6210\u529f";
  } catch (error) {
    authMessage.textContent = translateAuthError(error.message);
  } finally {
    button.disabled = false;
  }
});

accountHistoryList.addEventListener("click", (event) => {
  const button = event.target.closest("button[data-user-id]");
  if (!button) {
    return;
  }

  const userID = Number(button.dataset.userId);
  const account = readAccountHistory().find((item) => Number(item.id) === userID);
  if (!account) {
    renderAccountHistory();
    return;
  }

  rememberAccount(account);
  setCurrentUser(account);
  authMessage.textContent = "\u5df2\u8fdb\u5165\u5386\u53f2\u8d26\u53f7";
});

clearAccountHistoryButton.addEventListener("click", () => {
  localStorage.removeItem(accountHistoryKey);
  renderAccountHistory();
});

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  if (!currentUser) {
    authMessage.textContent = "\u8bf7\u5148\u767b\u5f55";
    usernameInput.focus();
    return;
  }

  const name = input.value.trim();
  if (!name) {
    message.textContent = "\u6e05\u5355\u540d\u79f0\u4e0d\u80fd\u4e3a\u7a7a";
    input.focus();
    return;
  }

  const button = form.querySelector(".enter-button");
  button.disabled = true;
  message.textContent = "\u6b63\u5728\u67e5\u770b\u6e05\u5355...";

  try {
    const room = await request("/api/rooms", {
      method: "POST",
      body: JSON.stringify({ name }),
    });

    rememberList(room);
    enterList(room);
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

  rememberList(room);
  enterList(room);
});

clearHistoryButton.addEventListener("click", () => {
  localStorage.removeItem(getHistoryKey());
  renderHistory();
});

logoutButton.addEventListener("click", () => {
  localStorage.removeItem(authKey);
  localStorage.removeItem(roomKey);
  currentUser = null;
  usernameInput.value = "";
  passwordInput.value = "";
  authMessage.textContent = "\u5df2\u9000\u51fa\u767b\u5f55";
  message.textContent = "";
  renderSignedOut();
});

function enterList(room) {
  localStorage.setItem(roomKey, JSON.stringify(room));
  window.location.href = `./tasks.html?roomId=${encodeURIComponent(room.id)}&roomName=${encodeURIComponent(room.name)}`;
}

function rememberAccount(user) {
  const account = {
    id: Number(user.id),
    username: user.username,
    token: user.token,
    loggedInAt: new Date().toISOString(),
  };
  const history = readAccountHistory();
  const nextHistory = [
    account,
    ...history.filter((item) => Number(item.id) !== account.id),
  ].slice(0, 8);

  localStorage.setItem(accountHistoryKey, JSON.stringify(nextHistory));
  renderAccountHistory();
}

function readAccountHistory() {
  try {
    const history = JSON.parse(localStorage.getItem(accountHistoryKey));
    return Array.isArray(history) ? history.filter((item) => item && item.id && item.username && item.token) : [];
  } catch (error) {
    localStorage.removeItem(accountHistoryKey);
    return [];
  }
}

function renderAccountHistory() {
  const history = readAccountHistory();
  clearAccountHistoryButton.disabled = history.length === 0;

  if (history.length === 0) {
    accountHistoryList.innerHTML = `<p class="history-empty">\u8fd8\u6ca1\u6709\u767b\u5f55\u8fc7\u7684\u8d26\u53f7</p>`;
    return;
  }

  accountHistoryList.innerHTML = history
    .map((account) => {
      const loggedInAt = account.loggedInAt ? formatVisitedAt(account.loggedInAt) : "\u6700\u8fd1";
      return `
        <button class="history-item" type="button" data-user-id="${account.id}">
          ${escapeHTML(account.username)}
          <span>${loggedInAt}</span>
        </button>
      `;
    })
    .join("");
}

function setCurrentUser(user) {
  currentUser = {
    id: Number(user.id),
    username: user.username,
    token: user.token,
  };
  localStorage.setItem(authKey, JSON.stringify(currentUser));
  renderSignedIn();
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

function renderSignedIn() {
  authForm.hidden = true;
  signedPanel.hidden = false;
  form.hidden = false;
  currentUsername.textContent = currentUser.username;
  renderHistory();
  input.focus();
}

function renderSignedOut() {
  authForm.hidden = false;
  signedPanel.hidden = true;
  form.hidden = true;
  currentUsername.textContent = "-";
  historyList.innerHTML = "";
  clearHistoryButton.disabled = true;
  renderAccountHistory();
  usernameInput.focus();
}

function rememberList(room) {
  const history = readHistory();
  const nextHistory = [
    {
      id: Number(room.id),
      name: room.name,
      visitedAt: new Date().toISOString(),
    },
    ...history.filter((item) => Number(item.id) !== Number(room.id)),
  ].slice(0, 6);

  localStorage.setItem(getHistoryKey(), JSON.stringify(nextHistory));
  renderHistory();
}

function readHistory() {
  if (!currentUser) {
    return [];
  }

  try {
    const history = JSON.parse(localStorage.getItem(getHistoryKey()));
    return Array.isArray(history) ? history.filter((item) => item && item.id && item.name) : [];
  } catch (error) {
    localStorage.removeItem(getHistoryKey());
    return [];
  }
}

function renderHistory() {
  const history = readHistory();
  clearHistoryButton.disabled = history.length === 0;

  if (history.length === 0) {
    historyList.innerHTML = `<p class="history-empty">\u8fd8\u6ca1\u6709\u67e5\u770b\u8fc7\u6e05\u5355</p>`;
    return;
  }

  historyList.innerHTML = history
    .map((room) => {
      const visitedAt = room.visitedAt ? formatVisitedAt(room.visitedAt) : "\u6700\u8fd1";
      return `
        <button class="history-item" type="button" data-room-id="${room.id}">
          ${escapeHTML(room.name)}
          <span>${visitedAt}</span>
        </button>
      `;
    })
    .join("");
}

function getHistoryKey() {
  return currentUser ? `taskListHistory:${currentUser.id}` : "taskListHistory";
}

function formatVisitedAt(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "\u6700\u8fd1";
  }

  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function translateAuthError(messageText) {
  const map = {
    "request body must be JSON": "\u8bf7\u6c42\u4f53\u5fc5\u987b\u662f JSON",
    "username is required": "\u7528\u6237\u540d\u4e0d\u80fd\u4e3a\u7a7a",
    "username cannot exceed 40 characters": "\u7528\u6237\u540d\u4e0d\u80fd\u8d85\u8fc7 40 \u4e2a\u5b57\u7b26",
    "password must be at least 6 characters": "\u5bc6\u7801\u81f3\u5c11\u9700\u8981 6 \u4f4d",
    "incorrect password": "\u5bc6\u7801\u4e0d\u6b63\u786e",
  };

  return map[messageText] || messageText;
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

async function init() {
  await loadConfig();
  loadCurrentUser();
  if (currentUser) {
    renderSignedIn();
    return;
  }
  renderSignedOut();
}

init();