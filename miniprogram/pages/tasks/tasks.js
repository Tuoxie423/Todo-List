const { request } = require("../../utils/request");
const roomHistory = require("../../utils/roomHistory");

const levels = ["基础", "进阶", "挑战"];
const listMeta = {
  learning: {
    titleKey: "learningTitle",
    levelIndexKey: "learningLevelIndex",
    added: "已添加学习任务",
  },
  optimization: {
    titleKey: "optimizationTitle",
    levelIndexKey: "optimizationLevelIndex",
    added: "已添加任务",
  },
};

Page({
  data: {
    room: null,
    roomName: "加载中",
    apiStatus: "连接中",
    apiOnline: false,
    levels,
    learningTitle: "",
    optimizationTitle: "",
    learningLevelIndex: 0,
    optimizationLevelIndex: 0,
    tasks: [],
    learningTasks: [],
    optimizationTasks: [],
    hasLearningTasks: false,
    hasOptimizationTasks: false,
    totalCount: 0,
    doneCount: 0,
    progressRate: 0,
    messages: {
      learning: "",
      optimization: "",
    },
  },

  onLoad(options) {
    const room = this.resolveRoom(options || {});
    if (!room) {
      wx.redirectTo({ url: "/pages/home/home" });
      return;
    }

    roomHistory.saveCurrentRoom(room);
    roomHistory.rememberRoom(room);
    this.setData({
      room,
      roomName: room.name,
    });
    this.checkHealth();
    this.loadTasks();
  },

  resolveRoom(options) {
    const roomID = Number(options.roomId);
    const roomName = decodeURIComponent(options.roomName || "");
    if (roomID > 0 && roomName) {
      return {
        id: roomID,
        name: roomName,
      };
    }

    return roomHistory.getCurrentRoom();
  },

  async checkHealth() {
    try {
      await request("/health");
      this.setData({
        apiStatus: "在线",
        apiOnline: true,
      });
    } catch (error) {
      this.setData({
        apiStatus: "未连接",
        apiOnline: false,
      });
    }
  },

  async loadTasks() {
    const room = this.data.room;
    if (!room) {
      return;
    }

    try {
      const data = await request(`/api/rooms/${room.id}/tasks`);
      this.updateTasks(data.items || []);
      this.setAllMessages("列表已更新");
    } catch (error) {
      this.setAllMessages(error.message || "无法读取任务");
    }
  },

  onTaskTitleInput(event) {
    const kind = event.currentTarget.dataset.kind;
    const meta = listMeta[kind];
    if (!meta) {
      return;
    }

    this.setData({
      [meta.titleKey]: event.detail.value,
    });
  },

  onLevelChange(event) {
    const kind = event.currentTarget.dataset.kind;
    const meta = listMeta[kind];
    if (!meta) {
      return;
    }

    this.setData({
      [meta.levelIndexKey]: Number(event.detail.value),
    });
  },

  async createTask(event) {
    const kind = event.currentTarget.dataset.kind;
    const meta = listMeta[kind];
    if (!meta || !this.data.room) {
      return;
    }

    const title = this.data[meta.titleKey].trim();
    const level = levels[this.data[meta.levelIndexKey]];
    if (!title) {
      this.setMessage(kind, "任务标题不能为空");
      return;
    }

    try {
      const task = await request(`/api/rooms/${this.data.room.id}/tasks`, {
        method: "POST",
        data: { title, level, kind },
      });

      this.updateTasks([task, ...this.data.tasks]);
      this.setData({ [meta.titleKey]: "" });
      this.setMessage(kind, meta.added);
    } catch (error) {
      this.setMessage(kind, error.message);
    }
  },

  async toggleTask(event) {
    const id = Number(event.currentTarget.dataset.id);
    const kind = event.currentTarget.dataset.kind;

    try {
      const updated = await request(`/api/rooms/${this.data.room.id}/tasks/${id}/toggle`, {
        method: "PATCH",
      });
      this.updateTasks(this.data.tasks.map((task) => (task.id === id ? updated : task)));
      this.setMessage(kind, updated.done ? "任务已完成" : "任务已恢复");
    } catch (error) {
      this.setMessage(kind, error.message);
    }
  },

  deleteTask(event) {
    const id = Number(event.currentTarget.dataset.id);
    const kind = event.currentTarget.dataset.kind;

    wx.showModal({
      title: "删除任务",
      content: "确定要删除这条任务吗？",
      confirmColor: "#e56f4a",
      success: async (result) => {
        if (!result.confirm) {
          return;
        }

        try {
          await request(`/api/rooms/${this.data.room.id}/tasks/${id}`, {
            method: "DELETE",
          });
          this.updateTasks(this.data.tasks.filter((task) => task.id !== id));
          this.setMessage(kind, "任务已删除");
        } catch (error) {
          this.setMessage(kind, error.message);
        }
      },
    });
  },

  changeRoom() {
    roomHistory.clearCurrentRoom();
    wx.redirectTo({ url: "/pages/home/home" });
  },

  updateTasks(tasks) {
    const normalizedTasks = tasks.map(formatTask);
    const doneCount = normalizedTasks.filter((task) => task.done).length;
    const totalCount = normalizedTasks.length;
    const progressRate = totalCount === 0 ? 0 : Math.round((doneCount / totalCount) * 100);

    const learningTasks = normalizedTasks.filter((task) => normalizeKind(task.kind) === "learning");
    const optimizationTasks = normalizedTasks.filter((task) => normalizeKind(task.kind) === "optimization");

    this.setData({
      tasks: normalizedTasks,
      learningTasks,
      optimizationTasks,
      hasLearningTasks: learningTasks.length > 0,
      hasOptimizationTasks: optimizationTasks.length > 0,
      totalCount,
      doneCount,
      progressRate,
    });
  },

  setMessage(kind, text) {
    this.setData({
      [`messages.${kind}`]: text,
    });
  },

  setAllMessages(text) {
    this.setData({
      messages: {
        learning: text,
        optimization: text,
      },
    });
  },
});

function formatTask(task) {
  return Object.assign({}, task, {
    kind: normalizeKind(task.kind),
    createdAtText: formatDate(task.createdAt),
  });
}

function normalizeKind(kind) {
  return kind === "optimization" ? "optimization" : "learning";
}

function formatDate(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "刚刚";
  }

  return `${pad(date.getMonth() + 1)}/${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function pad(value) {
  return String(value).padStart(2, "0");
}
