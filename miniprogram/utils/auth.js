const { request } = require("./request");

const authKey = "taskUser";

function getCurrentUser() {
  try {
    const user = wx.getStorageSync(authKey);
    if (user && Number(user.id) > 0 && user.username && user.token) {
      return {
        id: Number(user.id),
        username: user.username,
        token: user.token,
      };
    }
  } catch (error) {
    wx.removeStorageSync(authKey);
  }

  return null;
}

function saveCurrentUser(user) {
  const normalized = {
    id: Number(user.id),
    username: user.username,
    token: user.token,
  };
  wx.setStorageSync(authKey, normalized);
  return normalized;
}

function clearCurrentUser() {
  wx.removeStorageSync(authKey);
}

async function ensureLogin() {
  const savedUser = getCurrentUser();
  if (savedUser) {
    return savedUser;
  }

  const code = await wxLogin();
  const data = await request("/api/auth/wechat-login", {
    method: "POST",
    data: { code },
    skipAuth: true,
  });

  return saveCurrentUser(data.user);
}

function wxLogin() {
  return new Promise((resolve, reject) => {
    wx.login({
      success(result) {
        if (result.code) {
          resolve(result.code);
          return;
        }
        reject(new Error("微信登录失败"));
      },
      fail(error) {
        reject(new Error(error.errMsg || "微信登录失败"));
      },
    });
  });
}

module.exports = {
  authKey,
  getCurrentUser,
  saveCurrentUser,
  clearCurrentUser,
  ensureLogin,
};