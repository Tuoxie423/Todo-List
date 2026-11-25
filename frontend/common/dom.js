// 通用 DOM 辅助函数。
(function (global) {
  const Todo = (global.Todo = global.Todo || {});

  function escapeHTML(value) {
    return String(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#039;");
  }

  Todo.dom = { escapeHTML };
})(window);
