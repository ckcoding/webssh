# WebSSH

[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)

基于 Go + Vue 2 的 Web SSH / SFTP 工具，提供浏览器终端、主机管理、文件浏览与在线编辑能力。

## 截图

![终端预览](images/WX20260411-124130@2x.png)
![文件管理](images/WX20260411-124248@2x.png)

## 功能概览

- 浏览器终端：基于 WebSocket，支持 ANSI 色彩、特殊字符和多标签页。
- 主机管理：保存 SSH 主机配置，支持下次快速连接。
- 文件管理：支持目录浏览、上传、下载、新建和删除。
- 在线编辑：浏览器内编辑文本文件，支持 `Ctrl+S` / `Cmd+S` 保存。
- 基础监控：展示 CPU、内存、磁盘和系统信息。
- 登录鉴权：支持 `-a user:pass` 或 `authInfo=user:pass` 开启 Web 访问认证。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 前端 | Vue 2、Element UI、xterm.js |
| 后端 | Go、Gin、gorilla/websocket |
| 通信 | HTTP API、WebSocket |
| 远程能力 | SSH、SFTP |

## 部署前提

按部署方式准备环境：

| 场景 | 依赖 |
| --- | --- |
| Docker 部署 | Docker / Docker Compose |
| 宿主机部署 | Go 1.23+、Node.js 20+、npm |
| 本地一键脚本 | Go 1.23+、Node.js 20+、npm |

说明：

- 后端会把前端静态文件嵌入二进制，因此宿主机编译前必须先执行前端构建。
- 默认监听端口为 `5032`。
- 建议始终启用登录鉴权，不要直接暴露未鉴权实例到公网。

## 推荐部署：Docker

出于安全考虑，请自行从源码构建，不要直接使用来源不明的预构建镜像。

```bash
docker build --no-cache -f docker/Dockerfile -t webssh:local .

docker run -d \
  --name webssh \
  --restart unless-stopped \
  -p 5032:5032 \
  -e TZ=Asia/Shanghai \
  -e savePass=true \
  -e authInfo=admin:123456 \
  -v webssh-data:/home/webssh/.config/webssh \
  webssh:local
```

启动后访问 `http://127.0.0.1:5032/`。

### Docker Compose

仓库根目录已提供 [`docker-compose.yaml`](docker-compose.yaml)，可直接编译并启动：

```bash
docker compose up -d --build
```


### 环境变量

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `port` | 服务端口 | `5032` |
| `savePass` | 是否允许保存 SSH 密码 | `true` |
| `authInfo` | Web 登录认证，格式 `user:pass` | 未启用 |
| `TZ` | 时区 | `Asia/Shanghai` |

## 宿主机部署

```bash
cd web
npm ci
npm run build
cd ..

go build -trimpath -o webssh_server .
./webssh_server -a admin:123456 -p 5032 -s
```

可选参数：

```text
-a string  启用登录验证，格式为 user:pass
-p int     服务端口，默认 5032
-t int     SSH 会话超时分钟数，默认 120
-s         是否保存 SSH 密码，默认 true
-v         输出版本信息
```

## 安全建议

- 始终从当前仓库源码自行构建镜像或二进制。
- 生产环境务必启用 `-a user:pass` 登录保护。
- 建议放在 Nginx / Caddy 后面，通过 HTTPS 暴露。
- 不要使用来源不明的第三方镜像或可执行文件。

## 致谢

本项目基于 [Jrohy/webssh](https://github.com/Jrohy/webssh) 二次开发，遵循 [GPL-3.0](https://www.gnu.org/licenses/gpl-3.0.en.html) 开源协议。
