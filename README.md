# NetPulse

macOS 网络环境一键检测 & 优化工具。本地 Web 仪表盘，单二进制分发，双击即用。

## 功能

### 🔍 网络检测
- **Ping 测试** — 延迟、丢包率，支持自定义目标
- **DNS 测速** — 并发测试多个域名解析速度
- **路由追踪** — 可视化数据包路径，逐跳展示延迟
- **WiFi 分析** — SSID、信号强度(dBm)、信道、协议版本(802.11be/ax/ac)
- **配置查看** — 当前 DNS 服务器、HTTP/SOCKS 代理状态

### 🩺 智能诊断
一键检测 5 个维度（延迟、丢包、DNS、WiFi、代理），自动评分 0-100，给出：
- 问题严重程度 (🟢🟡🟠🔴)
- 问题原因说明
- 针对性修复建议
- 一键修复按钮

### ⚡ 一键优化
- **DNS 切换** — 5 个预设 (114DNS / 阿里 / DNSPod / Cloudflare / Google) + 自定义
- **DNS 缓存刷新** — sudo dscacheutil + killall mDNSResponder
- **代理开关** — HTTP/SOCKS 代理启停

## 截图

```
┌─────────────────────────────────────────────────────────┐
│  NetPulse  🟢 在线                         [刷新]       │
├─────────────────────────────────────────────────────────┤
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐ │
│  │ 📡 延迟      │ │ 🔍 DNS 速度  │ │ 📶 WiFi 信号     │ │
│  │ 0.4 ms       │ │ 2.6 ms       │ │ MyWiFi           │ │
│  │ 丢包率 0%    │ │ 114DNS       │ │ -44 dBm ▂▄▆█     │ │
│  └──────────────┘ └──────────────┘ └──────────────────┘ │
│                                                          │
│  ┌─ 深度检测 ───────────────────────────────────────┐   │
│  │ [Ping 百度] [Ping Google] [DNS 测速] [路由追踪]  │   │
│  │ [🩺 一键诊断]                                    │   │
│  │                                                   │   │
│  │ 🟢 评分 100  —  一切正常                          │   │
│  │ ✅ 延迟正常    Ping 延迟在合理范围内              │   │
│  │ ✅ 无丢包      网络连接稳定                       │   │
│  │ ✅ DNS 迅速    DNS 解析速度正常                   │   │
│  │ ✅ WiFi 良好   信号强度 (优秀)                    │   │
│  └───────────────────────────────────────────────────┘   │
│                                                          │
│  ┌─ 一键优化 ───────────────────────────────────────┐   │
│  │ DNS: [114DNS] [阿里DNS] [DNSPod] [Cloudflare]     │   │
│  │      [自定义: ________] [应用]                    │   │
│  │ [🚀 刷新 DNS 缓存]  [📡 代理开关]                 │   │
│  └───────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## 安装

### 下载二进制（推荐）

从 [Releases](../../releases) 下载对应平台的二进制文件，双击或命令行运行：

```bash
./netpulse
```

浏览器会自动打开 `http://127.0.0.1:8080`。

### 从源码编译

```bash
git clone https://github.com/你的用户名/netpulse.git
cd netpulse
make build        # 编译
./netpulse        # 运行
```

需要 Go 1.23+。

## 使用

```bash
./netpulse                # 启动（自动打开浏览器）
./netpulse -h             # 查看帮助
make run                  # 编译并运行
make build                # 仅编译
```

启动后浏览器访问 `http://127.0.0.1:8080`。

- 顶部 **状态卡片** 自动检测当前网络
- 左侧 **深度检测** — 手动触发 Ping、DNS、路由追踪、一键诊断
- 右侧 **一键优化** — 切换 DNS、刷新缓存、代理开关

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.23+ (net/http, embed) |
| 前端 | 原生 HTML/CSS/JS (零依赖，暗色主题) |
| 系统交互 | exec + macOS 系统命令 |
| 分发 | 单二进制 (~6MB)，`make release` 交叉编译 |

## API

```
GET  /api/status          # 快速状态总览
POST /api/detect/ping     # Ping 测试
POST /api/detect/dns      # DNS 解析测速
POST /api/detect/trace    # 路由追踪
GET  /api/detect/wifi     # WiFi 信息
GET  /api/detect/config   # DNS + 代理配置
GET  /api/diagnose        # 智能诊断
POST /api/optimize/dns    # 切换 DNS
POST /api/optimize/flush  # 刷新 DNS 缓存
POST /api/optimize/proxy  # 代理开关
```

## 项目结构

```
.
├── main.go            # 入口：HTTP 服务、端口探测、浏览器打开
├── handler.go         # API 路由 + JSON 响应
├── network.go         # 检测：Ping / DNS / 路由 / WiFi
├── optimize.go        # 优化：DNS 切换 / 缓存刷新 / 代理
├── diagnose.go        # 诊断引擎：分析 + 评分 + 建议
├── frontend/
│   └── index.html     # 单页 UI（暗色主题）
├── Makefile
├── LICENSE
└── README.md
```

## 系统要求

- **macOS** 10.15+（部分优化功能需要管理员权限）
- **Windows** 10/11（需管理员权限运行）
- **Linux**（需 NetworkManager / systemd-resolved / nmcli）

## 贡献

欢迎提 Issue 和 PR。请先讨论重大改动。

## 许可

MIT License © 2026 NetPulse Contributors
