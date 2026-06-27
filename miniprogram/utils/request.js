const config = require("./config");

function request(path, options) {
  const requestOptions = options || {};

  return new Promise((resolve, reject) => {
    wx.request({
      url: `${config.apiBase}${path}`,
      method: requestOptions.method || "GET",
      data: requestOptions.data || {},
      header: Object.assign({
        "Content-Type": "application/json",
      }, requestOptions.header || {}),
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

module.exports = {
  request,
  apiBase: config.apiBase,
};
