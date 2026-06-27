const { request } = require("../../utils/request");
const roomHistory = require("../../utils/roomHistory");

Page({
  data: {
    roomName: "",
    message: "",
    loading: false,
    history: [],
    hasHistory: false,
  },

  onShow() {
    this.refreshHistory();
  },

  onRoomNameInput(event) {
    this.setData({
      roomName: event.detail.value,
      message: "",
    });
  },

  async enterRoomSubmit() {
    const name = this.data.roomName.trim();
    if (!name) {
      this.setData({ message: "房间名称不能为空" });
      return;
    }

    this.setData({
      loading: true,
      message: "正在进入房间...",
    });

    try {
      const room = await request("/api/rooms", {
        method: "POST",
        data: { name },
      });
      this.openRoom(room);
    } catch (error) {
      this.setData({
        message: error.message,
        loading: false,
      });
    }
  },

  enterHistoryRoom(event) {
    const roomID = Number(event.currentTarget.dataset.id);
    const room = roomHistory.getHistory().find((item) => Number(item.id) === roomID);
    if (!room) {
      this.refreshHistory();
      return;
    }

    this.openRoom(room);
  },

  clearHistory() {
    roomHistory.clearHistory();
    this.refreshHistory();
  },

  openRoom(room) {
    const history = roomHistory.rememberRoom(room);
    roomHistory.saveCurrentRoom(room);
    this.setData({
      history: formatHistory(history),
      hasHistory: history.length > 0,
      loading: false,
      message: "",
    });

    wx.navigateTo({
      url: `/pages/tasks/tasks?roomId=${encodeURIComponent(room.id)}&roomName=${encodeURIComponent(room.name)}`,
    });
  },

  refreshHistory() {
    const history = roomHistory.getHistory();
    this.setData({
      history: formatHistory(history),
      hasHistory: history.length > 0,
      loading: false,
    });
  },
});

function formatHistory(history) {
  return history.map((room) => ({
    id: room.id,
    name: room.name,
    visitedAt: room.visitedAt,
    visitedAtText: formatVisitedAt(room.visitedAt),
  }));
}

function formatVisitedAt(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "最近";
  }

  const month = pad(date.getMonth() + 1);
  const day = pad(date.getDate());
  const hour = pad(date.getHours());
  const minute = pad(date.getMinutes());
  return `${month}/${day} ${hour}:${minute}`;
}

function pad(value) {
  return String(value).padStart(2, "0");
}
