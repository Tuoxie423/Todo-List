const auth = require("./auth");

const historyKey = "taskRoomHistory";
const currentRoomKey = "taskRoom";

function getHistory() {
  try {
    const history = wx.getStorageSync(userScopedKey(historyKey));
    return Array.isArray(history) ? history.filter(isValidRoom) : [];
  } catch (error) {
    wx.removeStorageSync(userScopedKey(historyKey));
    return [];
  }
}

function rememberRoom(room) {
  const normalizedRoom = normalizeRoom(room);
  if (!normalizedRoom) {
    return getHistory();
  }

  const history = getHistory();
  const nextHistory = [{
      id: normalizedRoom.id,
      name: normalizedRoom.name,
      visitedAt: new Date().toISOString(),
    }]
    .concat(history.filter((item) => Number(item.id) !== normalizedRoom.id))
    .slice(0, 6);

  wx.setStorageSync(userScopedKey(historyKey), nextHistory);
  return nextHistory;
}

function clearHistory() {
  wx.removeStorageSync(userScopedKey(historyKey));
}

function saveCurrentRoom(room) {
  const normalizedRoom = normalizeRoom(room);
  if (normalizedRoom) {
    wx.setStorageSync(userScopedKey(currentRoomKey), normalizedRoom);
  }
}

function getCurrentRoom() {
  try {
    const room = wx.getStorageSync(userScopedKey(currentRoomKey));
    return normalizeRoom(room);
  } catch (error) {
    wx.removeStorageSync(userScopedKey(currentRoomKey));
    return null;
  }
}

function clearCurrentRoom() {
  wx.removeStorageSync(userScopedKey(currentRoomKey));
}

function userScopedKey(key) {
  const user = auth.getCurrentUser();
  return user ? `${key}:${user.id}` : key;
}

function normalizeRoom(room) {
  if (!room) {
    return null;
  }

  const id = Number(room.id);
  const name = String(room.name || "").trim();
  if (!id || !name) {
    return null;
  }

  return { id, name };
}

function isValidRoom(room) {
  return Boolean(normalizeRoom(room));
}

module.exports = {
  getHistory,
  rememberRoom,
  rememberList: rememberRoom,
  clearHistory,
  saveCurrentRoom,
  saveCurrentList: saveCurrentRoom,
  getCurrentRoom,
  getCurrentList: getCurrentRoom,
  clearCurrentRoom,
  clearCurrentList: clearCurrentRoom,
};