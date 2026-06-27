const historyKey = "taskRoomHistory";
const currentRoomKey = "taskRoom";

function getHistory() {
  try {
    const history = wx.getStorageSync(historyKey);
    return Array.isArray(history) ? history.filter(isValidRoom) : [];
  } catch (error) {
    wx.removeStorageSync(historyKey);
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

  wx.setStorageSync(historyKey, nextHistory);
  return nextHistory;
}

function clearHistory() {
  wx.removeStorageSync(historyKey);
}

function saveCurrentRoom(room) {
  const normalizedRoom = normalizeRoom(room);
  if (normalizedRoom) {
    wx.setStorageSync(currentRoomKey, normalizedRoom);
  }
}

function getCurrentRoom() {
  try {
    const room = wx.getStorageSync(currentRoomKey);
    return normalizeRoom(room);
  } catch (error) {
    wx.removeStorageSync(currentRoomKey);
    return null;
  }
}

function clearCurrentRoom() {
  wx.removeStorageSync(currentRoomKey);
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
  clearHistory,
  saveCurrentRoom,
  getCurrentRoom,
  clearCurrentRoom,
};
