const { api, storage, dom } = window.Todo;

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

let currentUser = null;

authForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  const username = usernameInput.value.trim();
  const password = passwordInput.value;
  if (!username) {
    authMessage.textContent = "用户名不能为空";
    usernameInput.focus();
    return;
  }
  if (password.length < 6) {
    authMessage.textContent = "密码至少需要 6 位";
    passwordInput.focus();
    return;
  }

  const button = authForm.querySelector(".enter-button");
  button.disabled = true;
  authMessage.textContent = "正在登录...";

  try {
    const data = await api.request("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });

    rememberAccount(data.user);
    setCurrentUser(data.user);
    passwordInput.value = "";
    authMessage.textContent = data.registered ? "新用户已自动注册" : "登录成功";
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
  const account = storage.getAccountHistory().find((item) => Number(item.id) === userID);
  if (!account) {
    renderAccountHistory();
    return;
  }

  rememberAccount(account);
  setCurrentUser(account);
  authMessage.textContent = "已进入历史账号";
});

clearAccountHistoryButton.addEventListener("click", () => {
  storage.clearAccountHistory();
  renderAccountHistory();
});

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  if (!currentUser) {
    authMessage.textContent = "请先登录";
    usernameInput.focus();
    return;
  }

  const name = input.value.trim();
  if (!name) {
    message.textContent = "清单名称不能为空";
    input.focus();
    return;
  }

  const button = form.querySelector(".enter-button");
  button.disabled = true;
  message.textContent = "正在查看清单...";

  try {
    const room = await api.request("/api/rooms", {
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
  const room = storage.getRoomHistory().find((item) => Number(item.id) === roomID);
  if (!room) {
    renderHistory();
    return;
  }

  rememberList(room);
  enterList(room);
});

clearHistoryButton.addEventListener("click", () => {
  storage.clearRoomHistory();
  renderHistory();
});

logoutButton.addEventListener("click", () => {
  storage.clearCurrentUser();
  storage.clearCurrentRoom();
  currentUser = null;
  usernameInput.value = "";
  passwordInput.value = "";
  authMessage.textContent = "已退出登录";
  message.textContent = "";
  renderSignedOut();
});

function enterList(room) {
  storage.saveCurrentRoom(room);
  window.location.href = `./tasks.html?roomId=${encodeURIComponent(room.id)}&roomName=${encodeURIComponent(room.name)}`;
}

function rememberAccount(user) {
  storage.rememberAccount(user);
  renderAccountHistory();
}

function renderAccountHistory() {
  const history = storage.getAccountHistory();
  clearAccountHistoryButton.disabled = history.length === 0;

  if (history.length === 0) {
    accountHistoryList.innerHTML = `<p class="history-empty">还没有登录过的账号</p>`;
    return;
  }

  accountHistoryList.innerHTML = history
    .map((account) => {
      const loggedInAt = account.loggedInAt ? formatVisitedAt(account.loggedInAt) : "最近";
      return `
        <button class="history-item" type="button" data-user-id="${account.id}">
          ${dom.escapeHTML(account.username)}
          <span>${loggedInAt}</span>
        </button>
      `;
    })
    .join("");
}

function setCurrentUser(user) {
  currentUser = storage.saveCurrentUser(user);
  renderSignedIn();
}

function loadCurrentUser() {
  currentUser = storage.getCurrentUser();
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
  storage.rememberRoom(room);
  renderHistory();
}

function renderHistory() {
  const history = storage.getRoomHistory();
  clearHistoryButton.disabled = history.length === 0;

  if (history.length === 0) {
    historyList.innerHTML = `<p class="history-empty">还没有查看过清单</p>`;
    return;
  }

  historyList.innerHTML = history
    .map((room) => {
      const visitedAt = room.visitedAt ? formatVisitedAt(room.visitedAt) : "最近";
      return `
        <button class="history-item" type="button" data-room-id="${room.id}">
          ${dom.escapeHTML(room.name)}
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

function translateAuthError(messageText) {
  const map = {
    "request body must be JSON": "请求体必须是 JSON",
    "username is required": "用户名不能为空",
    "username cannot exceed 40 characters": "用户名不能超过 40 个字符",
    "password must be at least 6 characters": "密码至少需要 6 位",
    "incorrect password": "密码不正确",
  };

  return map[messageText] || messageText;
}

async function init() {
  await api.loadConfig();
  const notice = storage.consumeLoginNotice();
  loadCurrentUser();
  if (currentUser) {
    renderSignedIn();
    return;
  }
  renderSignedOut();
  if (notice) {
    authMessage.textContent = notice;
  }
}

init();
