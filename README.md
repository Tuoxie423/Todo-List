# Todo-list

一个用于学习 Gin 后端开发的多端任务清单项目。用户通过房间名称进入独立任务空间，管理「学习清单」和「任务清单」。

同一套 Gin API 同时支持浏览器端和微信小程序端。

## 功能

- 房间创建/进入
- 房间任务隔离
- 学习清单、任务清单管理
- 新增、完成/恢复、删除任务
- 浏览器端和小程序端历史房间记录
- YAML 配置端口和数据库连接

## 技术栈

- Backend：Go、Gin、GORM、MySQL
- Web：HTML、CSS、JavaScript
- Mini Program：微信小程序原生开发

## 目录

```text
backend/       Gin 后端
frontend/      浏览器端
miniprogram/   微信小程序端
config/        配置示例
```

## 快速启动

创建数据库：

```sql
CREATE DATABASE list DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

复制配置：

```bash
cp config/config.example.yaml config/config.yaml
```

修改 `config/config.yaml`：

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

启动后端：

```bash
cd backend
go mod tidy
go run .
```

启动浏览器端：

```bash
cd frontend
node server.js
```

访问：

```text
http://localhost:18090
```

小程序端用微信开发者工具打开 `miniprogram/` 目录。

## API

```text
POST   /api/rooms
GET    /api/rooms/:roomID/tasks
POST   /api/rooms/:roomID/tasks
PATCH  /api/rooms/:roomID/tasks/:taskID/toggle
DELETE /api/rooms/:roomID/tasks/:taskID
```

`kind` 可选值：

```text
learning      学习清单
optimization 任务清单
```

## 测试和构建

```bash
cd backend
go test ./...
go build -buildvcs=false -o todo-list .
```

```bash
node --check frontend/server.js
node --check frontend/home.js
node --check frontend/app.js
```

## 部署

推荐使用 Nginx 反向代理：

```text
/       -> 127.0.0.1:18090
/api/   -> 127.0.0.1:18080
/health -> 127.0.0.1:18080
```

线上环境把前端 API 地址改为 HTTPS 域名，例如：

```yaml
frontend:
  apiBase: https://list.tuoxie.asia
```

小程序接口地址在：

```text
miniprogram/utils/config.js
```

## 开源注意

不要提交：

- `config/config.yaml`
- 数据库账号密码
- 后端编译产物
- `miniprogram/project.private.config.json`

## License

建议按需补充开源协议，例如 MIT License。
