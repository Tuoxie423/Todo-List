const auth = require("../../utils/auth");
const { request } = require("../../utils/request");
const roomHistory = require("../../utils/roomHistory");

const SHARE_TITLE = "这个工具挺实用，推荐你试试";
const levels = ["今天", "本周", "近期", "有空"];
const urgencyClassMap = {
  今天: "today",
  本周: "week",
  近期: "soon",
  有空: "later",
};
const urgencyRankMap = {
  今天: 0,
  本周: 1,
  近期: 2,
  有空: 3,
};

Page({
  data: {
    room: null,
    roomName: "加载中",
    apiStatus: "连接中",
    apiOnline: false,
    levels,
    taskTitle: "",
    taskLevelIndex: 0,
    tasks: [],
    hasTasks: false,
    totalCount: 0,
    doneCount: 0,
    progressRate: 0,
    message: "",
  },

  onLoad(options) {
    wx.showShareMenu({
      withShareTicket: true,
      menus: ["shareAppMessage", "shareTimeline"],
    });


    const room = this.resolveList(options || {});
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

  resolveList(options) {
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
      this.setMessage("列表已更新");
    } catch (error) {
      this.setMessage(error.message || "无法读取任务");
    }
  },

  onTaskTitleInput(event) {
    this.setData({
      taskTitle: event.detail.value,
    });
  },

  onLevelChange(event) {
    this.setData({
      taskLevelIndex: Number(event.detail.value),
    });
  },

  async createTask() {
    if (!this.data.room) {
      return;
    }

    const title = this.data.taskTitle.trim();
    const level = levels[this.data.taskLevelIndex];
    if (!title) {
      this.setMessage("任务标题不能为空");
      return;
    }

    try {
      const task = await request(`/api/rooms/${this.data.room.id}/tasks`, {
        method: "POST",
        data: { title, level },
      });

      this.updateTasks([task, ...this.data.tasks]);
      this.setData({ taskTitle: "" });
      this.setMessage("已添加任务");
    } catch (error) {
      this.setMessage(error.message);
    }
  },

  async toggleTask(event) {
    const id = Number(event.currentTarget.dataset.id);

    try {
      const updated = await request(`/api/rooms/${this.data.room.id}/tasks/${id}/toggle`, {
        method: "PATCH",
      });
      this.updateTasks(this.data.tasks.map((task) => (task.id === id ? updated : task)));
      this.setMessage(updated.done ? "任务已完成" : "任务已恢复");
    } catch (error) {
      this.setMessage(error.message);
    }
  },

  deleteTask(event) {
    const id = Number(event.currentTarget.dataset.id);

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
          this.setMessage("任务已删除");
        } catch (error) {
          this.setMessage(error.message);
        }
      },
    });
  },

  changeList() {
    roomHistory.clearCurrentRoom();
    wx.redirectTo({ url: "/pages/home/home" });
  },

  updateTasks(tasks) {
    const normalizedTasks = sortTasks(tasks.map(formatTask));
    const doneCount = normalizedTasks.filter((task) => task.done).length;
    const totalCount = normalizedTasks.length;
    const progressRate = totalCount === 0 ? 0 : Math.round((doneCount / totalCount) * 100);

    this.setData({
      tasks: normalizedTasks,
      hasTasks: normalizedTasks.length > 0,
      totalCount,
      doneCount,
      progressRate,
    });
  },

  setMessage(text) {
    this.setData({
      message: text,
    });
  },

  onShareAppMessage() {
    return {
      title: SHARE_TITLE,
      path: buildSharePath(this.data.room),
    };
  },

  onShareTimeline() {
    const room = this.data.room;
    const query = room
      ? "roomId=" + encodeURIComponent(room.id) + "&roomName=" + encodeURIComponent(room.name)
      : "";

    return {
      title: SHARE_TITLE,
      query,
    };
  },
});

function buildSharePath(room) {
  if (!room) {
    return "/pages/home/home";
  }

  return "/pages/tasks/tasks?roomId=" + encodeURIComponent(room.id) + "&roomName=" + encodeURIComponent(room.name);
}

function formatTask(task) {
  const level = normalizeUrgency(task.level);
  return Object.assign({}, task, {
    level,
    urgencyClass: urgencyClassMap[level],
    createdAtText: formatDate(task.createdAt),
  });
}

function normalizeUrgency(level) {
  return levels.indexOf(level) >= 0 ? level : "有空";
}

function sortTasks(items) {
  return items.sort((left, right) => {
    const leftRank = urgencyRankMap[normalizeUrgency(left.level)];
    const rightRank = urgencyRankMap[normalizeUrgency(right.level)];
    if (leftRank !== rightRank) {
      return leftRank - rightRank;
    }

    const leftTime = new Date(left.createdAt).getTime() || 0;
    const rightTime = new Date(right.createdAt).getTime() || 0;
    return rightTime - leftTime;
  });
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
