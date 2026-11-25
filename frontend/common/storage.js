// 网页端唯一的 localStorage 读写入口：当前用户、账号历史、当前清单、清单历史。
(function (global) {
  const Todo = (global.Todo = global.Todo || {});

  const authKey = "taskUser";
  const accountHistoryKey = "taskAccountHistory";
  const roomKey = "taskList";
  const roomHistoryPrefix = "taskListHistory";
  const loginNoticeKey = "taskLoginNotice";
  const maxAccounts = 8;
  const maxRooms = 6;

  function normalizeUser(user) {
    if (!user) {
      return null;
    }

    const id = Number(user.id);
    const username = String(user.username || "").trim();
    const token = String(user.token || "");
    if (id > 0 && username && token) {
      return { id, username, token };
    }
    return null;
  }

  function normalizeRoom(room) {
    if (!room) {
      return null;
    }

    const id = Number(room.id);
    const name = String(room.name || "").trim();
    if (id > 0 && name) {
      return { id, name };
    }
    return null;
  }

  function readArray(key) {
    try {
      const value = JSON.parse(localStorage.getItem(key));
      return Array.isArray(value) ? value : [];
    } catch (error) {
      localStorage.removeItem(key);
      return [];
    }
  }

  function writeArray(key, value) {
    localStorage.setItem(key, JSON.stringify(value));
    return value;
  }

  function getCurrentUser() {
    try {
      return normalizeUser(JSON.parse(localStorage.getItem(authKey)));
    } catch (error) {
      localStorage.removeItem(authKey);
      return null;
    }
  }

  function saveCurrentUser(user) {
    const normalized = normalizeUser(user);
    if (normalized) {
      localStorage.setItem(authKey, JSON.stringify(normalized));
    }
    return normalized;
  }

  function clearCurrentUser() {
    localStorage.removeItem(authKey);
  }

  function getAccountHistory() {
    return readArray(accountHistoryKey).filter((item) => normalizeUser(item));
  }

  function rememberAccount(user) {
    const normalized = normalizeUser(user);
    if (!normalized) {
      return getAccountHistory();
    }

    const account = Object.assign({}, normalized, { loggedInAt: new Date().toISOString() });
    const next = [account]
      .concat(getAccountHistory().filter((item) => Number(item.id) !== account.id))
      .slice(0, maxAccounts);

    return writeArray(accountHistoryKey, next);
  }

  function forgetAccount(userID) {
    const target = Number(userID);
    writeArray(
      accountHistoryKey,
      getAccountHistory().filter((item) => Number(item.id) !== target)
    );
  }

  function clearAccountHistory() {
    localStorage.removeItem(accountHistoryKey);
  }

  // 登录过期提示用 sessionStorage 暂存，跳转后由首页读取并清除。
  function setLoginNotice(text) {
    try {
      sessionStorage.setItem(loginNoticeKey, text);
    } catch (error) {
      // 隐私模式下 sessionStorage 可能不可用，忽略即可。
    }
  }

  function consumeLoginNotice() {
    try {
      const text = sessionStorage.getItem(loginNoticeKey);
      sessionStorage.removeItem(loginNoticeKey);
      return text || "";
    } catch (error) {
      return "";
    }
  }

  function getCurrentRoom() {
    try {
      return normalizeRoom(JSON.parse(localStorage.getItem(roomKey)));
    } catch (error) {
      localStorage.removeItem(roomKey);
      return null;
    }
  }

  function saveCurrentRoom(room) {
    const normalized = normalizeRoom(room);
    if (normalized) {
      localStorage.setItem(roomKey, JSON.stringify(normalized));
    }
    return normalized;
  }

  function clearCurrentRoom() {
    localStorage.removeItem(roomKey);
  }

  function roomHistoryKey() {
    const user = getCurrentUser();
    return user ? `${roomHistoryPrefix}:${user.id}` : roomHistoryPrefix;
  }

  function getRoomHistory() {
    if (!getCurrentUser()) {
      return [];
    }
    return readArray(roomHistoryKey()).filter((item) => normalizeRoom(item));
  }

  function rememberRoom(room) {
    const normalized = normalizeRoom(room);
    if (!normalized) {
      return getRoomHistory();
    }

    const entry = Object.assign({}, normalized, { visitedAt: new Date().toISOString() });
    const next = [entry]
      .concat(getRoomHistory().filter((item) => Number(item.id) !== normalized.id))
      .slice(0, maxRooms);

    return writeArray(roomHistoryKey(), next);
  }

  function clearRoomHistory() {
    localStorage.removeItem(roomHistoryKey());
  }

  Todo.storage = {
    getCurrentUser,
    saveCurrentUser,
    clearCurrentUser,
    getAccountHistory,
    rememberAccount,
    forgetAccount,
    clearAccountHistory,
    setLoginNotice,
    consumeLoginNotice,
    getCurrentRoom,
    saveCurrentRoom,
    clearCurrentRoom,
    getRoomHistory,
    rememberRoom,
    clearRoomHistory,
  };
})(window);
