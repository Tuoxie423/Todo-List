const config = require("./config");

const authKey = "taskUser";

function request(path, options) {
  const requestOptions = options || {};
  const header = Object.assign({
    "Content-Type": "application/json",
  }, requestOptions.header || {});

  if (!requestOptions.skipAuth) {
    const user = getStoredUser();
    if (user && user.token) {
      header.Authorization = `Bearer ${user.token}`;
    }
  }

  return new Promise((resolve, reject) => {
    wx.request({
      url: `${config.apiBase}${path}`,
      method: requestOptions.method || "GET",
      data: requestOptions.data || {},
      header,
      success(response) {
        const statusCode = response.statusCode;

        if (statusCode >= 200 && statusCode < 300) {
          resolve(statusCode === 204 ? null : response.data);
          return;
        }

        const data = response.data || {};
        reject(new Error(data.message || `请求失败 (${statusCode})`));
      },
      fail(error) {
        reject(new Error(error.errMsg || "网络请求失败"));
      },
    });
  });
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