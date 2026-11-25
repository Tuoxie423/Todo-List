// 任务紧迫度的展示与排序规则，小程序端只在这里维护一份。
const levels = ["今天", "本周", "近期", "有空"];
const fallback = "有空";
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

function normalize(level) {
  return levels.indexOf(level) >= 0 ? level : fallback;
}

function classFor(level) {
  return urgencyClassMap[normalize(level)];
}

function rankOf(level) {
  return urgencyRankMap[normalize(level)];
}

function sortTasks(items) {
  return items.sort((left, right) => {
    const rankDiff = rankOf(left.level) - rankOf(right.level);
    if (rankDiff !== 0) {
      return rankDiff;
    }

    const leftTime = new Date(left.createdAt).getTime() || 0;
    const rightTime = new Date(right.createdAt).getTime() || 0;
    return rightTime - leftTime;
  });
}

module.exports = {
  levels,
  fallback,
  normalize,
  classFor,
  rankOf,
  sortTasks,
};
