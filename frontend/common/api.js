// 网页端统一的 API 客户端：读取 /config.json 并携带当前用户令牌。
(function (global) {
  const Todo = (global.Todo = global.Todo || {});

  const defaultApiBase = "http://localhost:18080";
  let apiBase = defaultApiBase;
  let redirectingToLogin = false;

  // 令牌失效时清掉本地凭证并回首页，避免页面显示"已登录"却加载不出数据。
  function handleExpiredSession(user) {
    if (redirectingToLogin) {
      return;
    }
    redirectingToLogin = true;

    Todo.storage.clearCurrentUser();
    Todo.storage.clearCurrentRoom();
    Todo.storage.forgetAccount(user.id);
    Todo.storage.setLoginNotice("登录已过期，请重新登录");
    window.location.href = "./home.html";
  }

  async function loadConfig() {
    try {
      const response = await fetch("/config.json");
      if (!response.ok) {
        throw new Error("config request failed");
      }

      const config = await response.json();
      apiBase = config.apiBase || defaultApiBase;
    } catch (error) {
      console.warn("Using default API base:", apiBase);
    }

    return apiBase;
  }

  function getApiBase() {
    return apiBase;
  }

  async function request(path, options = {}) {
    const headers = { "Content-Type": "application/json", ...options.headers };
    const user = Todo.storage.getCurrentUser();
    if (user) {
      headers.Authorization = `Bearer ${user.token}`;
    }

    const response = await fetch(`${apiBase}${path}`, { ...options, headers });
    if (response.status === 401 && user && !path.startsWith("/api/auth/")) {
      handleExpiredSession(user);
      throw new Error("登录已过期，请重新登录");
    }
    if (response.status === 204) {
      return null;
    }

    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(data.message || "请求失败");
    }

    return data;
  }

  Todo.api = { loadConfig, getApiBase, defaultApiBase, request };
})(window);
