const fs = require("fs");
const http = require("http");
const path = require("path");

const root = __dirname;
const configPath = process.env.CONFIG_PATH || path.join(root, "..", "config", "config.yaml");
const config = readConfig(configPath);
const frontendConfig = config.frontend || {};
const port = normalizePort(process.env.FRONTEND_PORT || frontendConfig.port || 5173);
const browserConfig = {
  apiBase: process.env.API_BASE || frontendConfig.apiBase || "http://localhost:18080",
};
const types = {
  ".html": "text/html; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".png": "image/png",
  ".svg": "image/svg+xml; charset=utf-8",
  ".ico": "image/x-icon",
};

const server = http.createServer((req, res) => {
  try {
    const urlPath = getURLPath(req);

    if (urlPath === "/config.json") {
      res.writeHead(200, { "Content-Type": "application/json; charset=utf-8" });
      res.end(JSON.stringify(browserConfig));
      return;
    }

    const filePath = path.join(root, urlPath === "/" ? "index.html" : urlPath);

    if (!isInsideRoot(filePath)) {
      res.writeHead(403);
      res.end("Forbidden");
      return;
    }

    fs.readFile(filePath, (err, content) => {
      if (err) {
        res.writeHead(404);
        res.end("Not found");
        return;
      }

      res.writeHead(200, {
        "Content-Type": types[path.extname(filePath)] || "application/octet-stream",
      });
      res.end(content);
    });
  } catch (error) {
    console.error(`Request failed: ${error.message}`);
    if (!res.headersSent) {
      res.writeHead(error.statusCode || 500);
    }
    res.end(error.statusCode === 400 ? "Bad request" : "Internal server error");
  }
});

server.listen(port, () => {
  console.log(`Frontend: http://localhost:${port}`);
  console.log(`API base: ${browserConfig.apiBase}`);
});

server.on("error", (error) => {
  console.error(`Frontend server failed: ${error.message}`);
  process.exit(1);
});

function normalizePort(value) {
  const normalized = Number(value);
  if (!Number.isInteger(normalized) || normalized <= 0 || normalized > 65535) {
    throw new Error(`Invalid frontend port: ${value}`);
  }

  return normalized;
}

function readConfig(filePath) {
  try {
    return parseSimpleYAML(fs.readFileSync(resolveConfigFile(filePath), "utf8"));
  } catch (error) {
    console.warn(`Config not loaded: ${error.message}`);
    return {};
  }
}

function resolveConfigFile(filePath) {
  try {
    if (fs.statSync(filePath).isDirectory()) {
      return path.join(filePath, "config.yaml");
    }
  } catch (error) {
    if (![".yaml", ".yml"].includes(path.extname(filePath))) {
      return path.join(filePath, "config.yaml");
    }
  }

  return filePath;
}

function parseSimpleYAML(text) {
  const data = {};
  let section = null;

  for (const rawLine of text.split(/\r?\n/)) {
    const lineWithoutComment = rawLine.replace(/\s+#.*$/, "");
    if (!lineWithoutComment.trim()) {
      continue;
    }

    const indent = lineWithoutComment.match(/^\s*/)[0].length;
    const line = lineWithoutComment.trim();
    const colonIndex = line.indexOf(":");
    if (colonIndex === -1) {
      continue;
    }

    const key = line.slice(0, colonIndex).trim();
    const value = line.slice(colonIndex + 1).trim();

    if (indent === 0 && value === "") {
      data[key] = {};
      section = key;
      continue;
    }

    if (indent > 0 && section) {
      data[section][key] = parseScalar(value);
      continue;
    }

    data[key] = parseScalar(value);
    section = null;
  }

  return data;
}

function parseScalar(value) {
  if (value === "true") {
    return true;
  }
  if (value === "false") {
    return false;
  }
  if (value !== "" && !Number.isNaN(Number(value))) {
    return Number(value);
  }
  return value.replace(/^["']|["']$/g, "");
}

function getURLPath(req) {
  try {
    return decodeURIComponent(new URL(req.url, "http://localhost").pathname);
  } catch (error) {
    const badRequest = new Error("Invalid request URL");
    badRequest.statusCode = 400;
    throw badRequest;
  }
}

function isInsideRoot(filePath) {
  const relativePath = path.relative(root, filePath);
  return relativePath && !relativePath.startsWith("..") && !path.isAbsolute(relativePath);
}
