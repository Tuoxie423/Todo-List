const config = require("./config");

const authKey = "taskUser";

let reloginPromise = null;

async function request(path, options) {
  return send(path, options || {}, false);
}

async function send(path, requestOptions, retried) {
  const header = Object.assign({
    "Content-Type": "application/json",
  }, requestOptions.header || {});

  const user = requestOptions.skipAuth ? null : getStoredUser();
  if (user && user.token) {
    header.Authorization = `Bearer ${user.token}`;
  }

  const response = await wxRequest(path, requestOptions, header);
  const statusCode = response.statusCode;

  if (statusCode >= 200 && statusCode < 300) {
    return statusCode === 204 ? null : response.data;
  }

  // 令牌失效时静默重新登录并重试一次；登录类请求和已重试过的请求不再触发。
  if (statusCode === 401 && user && !retried && !path.startsWith("/api/auth/")) {
    await relogin();
    return send(path, requestOptions, true);
  }

  const data = response.data || {};
  throw new Error(data.message || `请求失败 (${statusCode})`);
}

function wxRequest(path, requestOptions, header) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${config.apiBase}${path}`,
      method: requestOptions.method || "GET",
      data: requestOptions.data || {},
      header,
      success: resolve,
      fail(error) {
        reject(new Error(error.errMsg || "网络请求失败"));
      },
    });
  });
}

// 同一时刻只发起一次微信重新登录，避免并发 401 互相覆盖新令牌。
function relogin() {
  if (!reloginPromise) {
    // 延迟 require，避免 request 与 auth 在加载阶段形成循环依赖。
    const auth = require("./auth");
    auth.clearCurrentUser();
    reloginPromise = auth.ensureLogin().then(
      (user) => {
        reloginPromise = null;
        return user;
      },
      (error) => {
        reloginPromise = null;
        throw error;
      }
    );
  }

  return reloginPromise;
}

function getStoredUser() {
  try {
    const user = wx.getStorageSync(authKey);
    return user && user.token ? user : null;
  } catch (error) {
    return null;
  }
}

module.exports = {
  request,
  apiBase: config.apiBase,
};
