const auth = require("../../utils/auth");
const { request } = require("../../utils/request");
const roomHistory = require("../../utils/roomHistory");

const SHARE_TITLE = "这个工具挺实用，推荐你试试";

Page({
  data: {
    user: null,
    loginLoading: true,
    roomName: "",
    message: "",
    loading: false,
    history: [],
    hasHistory: false,
  },

  onLoad() {
    wx.showShareMenu({
      withShareTicket: true,
      menus: ["shareAppMessage", "shareTimeline"],
    });
  },

  async onShow() {
    await this.ensureWechatLogin();
    this.refreshHistory();
  },

  async ensureWechatLogin() {
    this.setData({ loginLoading: true, message: "正在微信登录..." });

    try {
      const user = await auth.ensureLogin();
      this.setData({
        user,
        loginLoading: false,
        message: "微信登录成功",
      });
    } catch (error) {
      this.setData({
        user: null,
        loginLoading: false,
        message: error.message || "微信登录失败，请稍后重试",
      });
    }
  },

  async relogin() {
    auth.clearCurrentUser();
    roomHistory.clearCurrentList();
    await this.ensureWechatLogin();
    this.refreshHistory();
  },

  onListNameInput(event) {
    this.setData({
      roomName: event.detail.value,
      message: "",
    });
  },

  async enterListSubmit() {
    if (!this.data.user) {
      await this.ensureWechatLogin();
      if (!this.data.user) {
        return;
      }
    }

    const name = this.data.roomName.trim();
    if (!name) {
      this.setData({ message: "清单名称不能为空" });
      return;
    }

    this.setData({
      loading: true,
      message: "正在查看清单...",
    });

    try {
      const room = await request("/api/rooms", {
        method: "POST",
        data: { name },
      });
      this.openList(room);
    } catch (error) {
      this.setData({
        message: error.message,
        loading: false,
      });
    }
  },

  enterHistoryList(event) {
    const roomID = Number(event.currentTarget.dataset.id);
    const room = roomHistory.getHistory().find((item) => Number(item.id) === roomID);
    if (!room) {
      this.refreshHistory();
      return;
    }

    this.openList(room);
  },

  clearHistory() {
    roomHistory.clearHistory();
    this.refreshHistory();
  },

  openList(room) {
    const history = roomHistory.rememberList(room);
    roomHistory.saveCurrentList(room);
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

  onShareAppMessage() {
    return {
      title: SHARE_TITLE,
      path: "/pages/home/home",
    };
  },

  onShareTimeline() {
    return {
      title: SHARE_TITLE,
      query: "",
    };
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