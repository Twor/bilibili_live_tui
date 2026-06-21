# bilibili 直播间 TUI

终端界面的 Bilibili 直播工具，支持实时弹幕、发送消息、多主题切换、语音朗读、桌面通知。

## 功能

- 实时弹幕、SC、礼物、上舰、进入房间事件
- 发送弹幕到直播间（自动分段）
- 多主题切换（4 种布局）
- 桌面通知（弹幕/SC/礼物/上舰）
- **语音朗读**（SC/礼物/上舰优先，弹幕防堆积）
- **直播状态显示**（直播中/未开播/轮播）
- **礼物金额统计**（标题栏右侧累计显示）

## 快速开始

```bash
# 安装 edge-tts（语音朗读需要，可选）
pip install edge-tts

# 运行
go run .

# 指定房间号和主题
go run . -r 7777 -t 3

# 开启语音朗读
go run . -k 1

# 强制重新扫码登录
go run . --login

# 完整参数
go run . -c ~/.config/bili/config.toml -r 9527 -t 1 -k 1
```

## 扫码登录

首次运行会自动弹出二维码，使用 B 站 APP 扫码即可登录。登录信息自动保存，后续启动自动校验 Cookie 有效期，即将过期时自动刷新。

## 参数说明

| 参数 | 用途 | 默认值 |
|------|------|--------|
| `-c` | 配置文件路径 | `~/.config/bili/config.toml` |
| `-r` | 直播间 ID | 配置文件 |
| `-t` | 主题 (1-4) | 配置文件 |
| `-l` | 单行模式 (0/1) | 配置文件 |
| `-s` | 显示时间 (0/1) | 配置文件 |
| `-n` | 桌面通知 (0/1) | 配置文件 |
| `-k` / `--speak` | 语音朗读 (0/1) | 配置文件 |
| `--login` | 强制重新扫码登录 | false |

## 语音朗读

需要安装 [edge-tts](https://github.com/rany2/edge-tts)（Python 包，跨平台，调微软云端 TTS）：

```bash
pip install edge-tts
```

启动时加 `-k 1` 或在配置中设置 `Speak = 1`。

**优先级**：SC/礼物/上舰优先朗读，普通弹幕在队列积压超过 5 条时自动丢弃旧消息，只读最新一条。

| 事件 | 朗读文案 |
|------|---------|
| 普通弹幕 | `用户名说：内容` |
| SC | `Super Chat X元，来自用户名：内容` |
| 礼物 | `用户名 投喂了 N 个 礼物名` |
| 上舰 | `用户名 购买了 舰长名` |

> 语音朗读依赖 edge-tts，未安装时朗读功能不可用。

## 快捷键

| 键 | 功能 |
|----|------|
| `<Esc>` / `<Ctrl+C>` | 退出 |
| `<Ctrl+U>` | 清空输入 |
| `<Up>` / `<Down>` | 浏览发送历史 |
| `<Enter>` | 发送弹幕 |

## 主题

| 主题 | 名称 | 布局 |
|------|------|------|
| 1 | 聊天室 | 左侧边栏（房间信息+贡献榜）+ 右侧聊天+输入 |
| 2 | 极简 | 仅聊天+输入 |
| 3 | 简单 | 顶部标题栏（含直播状态+礼物总额）→ 消息 → 输入 |
| 4 | 信息 | 左侧边栏 + 弹幕/进门/礼物分流显示 |

## 配置

默认配置文件：`~/.config/bili/config.toml`

```toml
Cookie = ""
RoomId = 7777
Theme = 3
SingleLine = 1
ShowTime = 1
Notify = 1
Speak = 0
```

首次运行自动生成，颜色配置项可自定义。

## 项目结构

```
├── main.go          # 入口
├── config/          # 配置加载、登录生命周期
├── getter/          # 弹幕获取（WebSocket + HTTP 轮询）
│   ├── getter.go    # 直播客户端、事件处理器
│   ├── speaker.go   # 语音朗读（edge-tts + 优先级队列）
│   └── notify.go    # 桌面通知
├── login/           # 扫码登录、Cookie 管理
├── sender/          # 发送弹幕到直播间
└── ui/              # 终端 UI
    ├── common/      # 共享消息处理器
    ├── theme1/      # 聊天室主题
    ├── theme2/      # 极简主题
    ├── theme3/      # 简单主题
    └── theme4/      # 信息主题
```

## 类似项目

- [zaiic/bili-live-chat](https://github.com/zaiic/bili-live-chat) — Rust 实现的 B 站直播聊天 TUI

## 贡献者

- [yaocccc](https://github.com/yaocccc)
- [soft98-top](https://github.com/soft98-top) — theme4
- [zaiic](https://github.com/zaiic)
- [Ruixi-rebirth](https://github.com/Ruixi-rebirth) — 自动创建配置文件
