# Gin Task Room

一个用于学习 Gin 后端开发的多端任务清单项目。用户可以通过房间名称进入独立的任务空间，在同一个房间里管理「学习清单」和「任务清单」。

项目采用前后端分离结构，同一套 Gin API 同时支持浏览器端和微信小程序端。

## 功能特性

- 通过房间名称创建或进入任务房间
- 不同房间之间的任务数据相互隔离
- 支持学习清单和任务清单两个模块
- 支持新增任务、标记完成、恢复任务、删除任务
- 浏览器端记录最近进入过的历史房间
- 小程序端记录最近进入过的历史房间
- 使用 YAML 统一管理后端端口、前端端口、数据库连接等配置

## 项目结构

```text
.
├── backend                 # Gin 后端服务
│   ├── config              # 后端配置读取
│   ├── global              # 数据库连接
│   ├── model               # GORM 数据模型
│   ├── main.go             # 路由和接口入口
│   └── main_test.go        # 接口测试
├── config
│   └── config.example.yaml # 配置示例
├── frontend                # 浏览器端页面
│   ├── home.html           # 房间首页
│   ├── tasks.html          # 任务页
│   ├── app.js
│   ├── home.js
│   └── server.js           # 前端静态服务
├── miniprogram             # 微信小程序端
│   ├── pages
│   ├── utils
│   └── assets
└── README.md
```

## 技术栈

- 后端：Go、Gin、GORM
- 数据库：MySQL
- 浏览器端：原生 HTML、CSS、JavaScript
- 小程序端：微信小程序原生开发
- 配置管理：YAML

## 本地启动

### 1. 准备数据库

先创建 MySQL 数据库，例如：

```sql
CREATE DATABASE list DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2. 准备配置文件

复制示例配置：

```bash
cp config/config.example.yaml config/config.yaml
```

按本地环境修改 `config/config.yaml`：

```yaml
backend:
  port: 18080
  mode: release

frontend:
  port: 18090
  apiBase: http://localhost:18080

database:
  host: localhost
  port: 3306
  user: your_username
  password: your_password
  name: list
  charset: utf8mb4
```

`config/config.yaml` 包含数据库账号密码，已经加入 `.gitignore`，不要提交到仓库。

### 3. 启动后端

```bash
cd backend
go mod tidy
go run .
```

后端默认读取 `../config/config.yaml`，也可以通过 `CONFIG_PATH` 指定配置文件：

```bash
CONFIG_PATH=/path/to/config.yaml go run .
```

健康检查：

```bash
curl http://localhost:18080/health
```

### 4. 启动浏览器端

新开一个终端：

```bash
cd frontend
node server.js
```

访问：

```text
http://localhost:18090
```

前端服务也支持通过环境变量覆盖配置：

```bash
CONFIG_PATH=/path/to/config.yaml node server.js
```

也可以临时覆盖端口和 API 地址：

```bash
FRONTEND_PORT=18090 API_BASE=http://localhost:18080 node server.js
```

### 5. 启动小程序端

用微信开发者工具打开 `miniprogram/` 目录。

小程序接口地址在：

```text
miniprogram/utils/config.js
```

本地调试可改为：

```js
const config = {
  apiBase: "http://localhost:18080",
};
```

正式发布时改为你的 HTTPS 域名，例如：

```js
const config = {
  apiBase: "https://list.tuoxie.asia",
};
```

微信小程序正式版还需要在微信公众平台配置 request 合法域名。

## API 接口

```text
POST   /api/rooms
GET    /api/rooms/:roomID/tasks
POST   /api/rooms/:roomID/tasks
PATCH  /api/rooms/:roomID/tasks/:taskID/toggle
DELETE /api/rooms/:roomID/tasks/:taskID
```

### 创建或进入房间

```http
POST /api/rooms
Content-Type: application/json
```

```json
{
  "name": "Go 后端练习"
}
```

### 查询房间任务

```http
GET /api/rooms/1/tasks
```

### 创建任务

```http
POST /api/rooms/1/tasks
Content-Type: application/json
```

```json
{
  "title": "学习 Gin 路由分组",
  "level": "基础",
  "kind": "learning"
}
```

`kind` 可选值：

```text
learning      学习清单
optimization 任务清单
```

说明：后端内部仍使用 `optimization` 作为任务清单的分类值，这是为了保持数据兼容；前端展示名称是「任务清单」。

## 测试和构建

运行后端测试：

```bash
cd backend
go test ./...
```

构建后端：

```bash
cd backend
go build -buildvcs=false -o gin-practice .
```

检查前端脚本：

```bash
node --check frontend/server.js
node --check frontend/home.js
node --check frontend/app.js
```

检查小程序脚本：

```bash
node --check miniprogram/pages/home/home.js
node --check miniprogram/pages/tasks/tasks.js
node --check miniprogram/utils/request.js
node --check miniprogram/utils/roomHistory.js
```

## 部署说明

推荐使用 Nginx 对外提供 HTTPS，再反向代理到本机前后端服务：

```text
https://your-domain.com
        ↓
Nginx 443
        ↓
/       -> 127.0.0.1:18090
/api/   -> 127.0.0.1:18080
/health -> 127.0.0.1:18080
```

Nginx 示例：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    location /api/ {
        proxy_pass http://127.0.0.1:18080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /health {
        proxy_pass http://127.0.0.1:18080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        proxy_pass http://127.0.0.1:18090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

SSL 证书可以使用 Certbot 申请：

```bash
sudo certbot --nginx -d your-domain.com
```

## 开源注意事项

提交代码前建议检查：

```bash
git status
```

不要提交：

- `config/config.yaml`
- 数据库账号密码
- 后端编译产物，例如 `backend/gin-practice`
- Go 测试产物和缓存
- 微信开发者工具私有配置，例如 `miniprogram/project.private.config.json`

可以提交：

- `config/config.example.yaml`
- `backend/`
- `frontend/`
- `miniprogram/`
- `README.md`
- `.gitignore`

## 后续可扩展

- 用户登录
- 房间权限控制
- 任务编辑
- 任务分页
- 操作日志
- 更完整的接口测试
- Docker 部署

## License

如果准备正式开源，建议补充一个明确的开源协议文件，例如 MIT License。
