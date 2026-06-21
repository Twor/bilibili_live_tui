 # bilibili_live_tui — 项目指南

 ## 项目概述

 Bilibili 直播间 TUI 工具，用 Go 编写，基于 tview/tcell 渲染终端界面。
 功能：查看弹幕/SC/礼物/上舰/进入房间事件，发送弹幕，多主题切换，桌面通知。

 ## 架构概览

 ```
 main.go                      # 入口：初始化 channel → getter/sender/ui
 ├── config/                  # 配置加载、扫码登录生命周期、CLI 参数
 ├── getter/                  # 消息获取（WebSocket 实时弹幕 + HTTP 轮询房间信息）
 ├── sender/                  # 发送弹幕到 B 站直播间
 ├── login/                   # B 站扫码登录、Cookie 刷新/校验
 └── ui/                      # 终端 UI
     ├── common/              # 共享的消息处理 handler
     ├── theme1/              # 聊天室主题（侧边栏 + 聊天）
     ├── theme2/              # 极简主题（仅聊天）
     ├── theme3/              # 简单主题（信息栏 + 分隔线 + 聊天）
     └── theme4/              # 信息主题（侧边栏 + 分流弹幕/进门/礼物）
 ```

 ### 数据流

 1. `main.go` 创建两个带缓冲 channel：`busChan`（弹幕消息）和 `roomInfoChan`（房间信息）
 2. `config.Init()` 加载 `~/.config/bili/config.toml`，处理登录生命周期
 3. `getter.Run()` 启动三个 goroutine：
    - `syncRoomInfo` — 每 30 秒 HTTP 轮询房间信息和贡献榜
    - `getHistory` — 一次性拉取最近历史弹幕
    - `startBlivedmClient` — WebSocket 长连接接收实时事件
 4. `sender.Run()` 校验 Cookie、提取 CSRF token
 5. `ui.Run()` 根据配置选择主题渲染界面

 ### Channel 契约

 | Channel | 类型 | 容量 | 消费者 |
 |---------|------|------|--------|
 | `busChan` | `chan DanmuMsg` | 512 | `common.DanmuHandler` + theme4 分流 handler |
 | `roomInfoChan` | `chan RoomInfo` | 32 | `common.RoomInfoHandler` |

 所有 channel 写入使用 `select/default` 避免阻塞；关闭由 SIGINT/SIGTERM 触发。

 ## 编译与运行

 ```bash
 # 编译（二进制名由 go.mod 的 module 决定，为 "bili"）
 go build -o bili .

 # 直接运行
 go run .

 # 指定房间号和主题
 go run . -r 7777 -t 3

 # 强制重新扫码登录
 go run . --login

 # 完整参数
 go run . -c ~/.config/bili/config.toml -r 9527 -t 1 -l 1 -s 1 -n 1
 ```

 ### CLI 参数

 | 参数 | 用途 | 默认 |
 |------|------|------|
 | `-c` | 配置文件路径 | `~/.config/bili/config.toml` |
 | `-r` | 直播间 ID | 配置文件中读取 |
 | `-t` | 主题 (1-4) | 配置文件中读取 |
 | `-l` | 单行模式 (0/1) | 配置文件中读取 |
 | `-s` | 显示时间 (0/1) | 配置文件中读取 |
 | `-n` | 桌面通知 (0/1) | 配置文件中读取 |
 | `--login` | 强制重新扫码登录 | false |

 ### 快捷键

 - `<Esc>` / `<Ctrl+C>` — 退出
 - `<Ctrl+U>` — 清空输入
 - `<Up>` / `<Down>` — 浏览发送历史
 - `<Enter>` — 发送弹幕

 ## 包指南

 ### `config/` — 配置与登录生命周期

 - 首次运行自动生成 `~/.config/bili/config.toml`
 - 启动时自动检查 Cookie 有效期，7 天内过期自动刷新，已过期则触发登录
 - `SaveConfig()` 将当前配置写回文件
 - 颜色字段为空时会统一设置默认值 `#bbbbbb`
 - **添加配置项**：在 `ConfigType` 结构体加字段，`defaultConfig()` 设默认值，`flag.StringVar/Int64Var` 加 CLI 参数

 ### `getter/` — 消息获取

 - `blivedm-go` 提供 WebSocket 连接，自动重连
 - 事件类型：`DANMU_MSG`、`SUPER_CHAT`、`SEND_GIFT`、`GUARD_BUY`、`INTERACT_WORD`
 - 房间信息通过 HTTP API `room/v1/Room/get_info` 获取
 - 贡献榜通过 HTTP API `xlive/general-interface/v1/rank/getOnlineGoldRank` 获取
 - 历史弹幕通过 HTTP API `xlive/web-room/v1/dM/gethistory` 获取（仅首次）
 - `DanmuMsg` 类型包含：消息类型、作者、内容、时间、粉丝牌、礼物信息
 - `notify.go` 使用 `notify-send` 发送桌面通知，忽略执行错误

 ### `login/` — 扫码登录

 - 使用 B 站 passport 接口生成二维码和轮询扫码状态
 - QR 码用 `go-qrcode` 生成，缩小版 Unicode 半块字符输出
 - 登录成功获取 Cookie（含 `bili_jct`）和 RefreshToken
 - `VerifyCookie` 通过 `x/web-interface/nav` 接口校验登录状态
 - `RefreshCookie` 通过访问 nav 接口刷新 Cookie 有效期

 ### `sender/` — 发送弹幕

 - 调用 B 站 `msg/send` API 发送弹幕
 - 超过 20 字符自动分段发送，间隔 1 秒
 - 依赖 `bili_jct`（CSRF token）从 Cookie 中提取
 - 发送失败通过 `busChan` 回显系统通知

 ### `ui/` — 终端界面

 #### 主题系统

 每个主题是一个独立 package，暴露 `Run(busChan, roomInfoChan)` 函数。
 `ui/ui.go` 根据 `Config.Theme` 路由到对应主题。

 | 主题 | 包 | 布局 |
 |------|-----|------|
 | 1 — 聊天室 | `theme1` | 左侧边栏（房间信息+贡献榜）+ 右侧聊天+输入 |
 | 2 — 极简 | `theme2` | 仅聊天+输入，无边框 |
 | 3 — 简单 | `theme3` | 顶部房间信息 → 分隔线 → 聊天 → 分隔线 → 输入 |
 | 4 — 信息 | `theme4` | 左侧边栏 + 右侧三栏（弹幕/进门/礼物事件分流） |

 #### `ui/common/` — 共享 Handler

 - `DanmuHandler` — 格式化并输出弹幕/SC/礼物/上舰/进门消息到 `TextView`
 - `RoomInfoHandler` — 格式化房间信息和贡献榜到 `TextView`
 - `SetBoxAttr` — 统一设置 tview Box 的边框、标题、颜色
 - `SubmitHistory` — 全局发送历史（最多 10 条），支持上下键导航

 #### 添加新主题

 1. 新建 `ui/themeN/` 目录
 2. 实现 `Run(busChan chan getter.DanmuMsg, roomInfoChan chan getter.RoomInfo)`
 3. 在 `ui/ui.go` 的 `switch` 中注册新 case
 4. 更新 `AGENTS.md` 的主题表格

 #### tview 使用约定

 - 所有 `TextView` 调用 `SetDynamicColors(true)` 以支持颜色标签
 - 背景色统一通过 `common.Bg` 设置
 - 输入框使用 `SetFormAttributes` 设置颜色
 - Grid 布局优先，不启用鼠标事件
 - theme1/theme4 使用 `SetColumns()` 固定侧边栏宽度

 ## 代码约定

 ### Go 风格

 - 使用英文标识符，注释用中文（项目已有风格）
 - 错误处理使用 `fmt.Errorf("上下文: %w", err)` 包装
 - goroutine 启动在调用方，不在被调用函数内部隐藏启动
 - channel 写入使用 `select/default` 避免阻塞
 - HTTP 客户端复用 `&http.Client{Timeout: N*time.Second}` 实例

 ### 消息颜色格式

 tview 颜色标签格式：`[#RRGGBB]text[#RRGGBB]`，全局颜色从 `config.Config` 读取。

 | 消息类型 | 固定颜色 |
 |----------|---------|
 | 弹幕（有粉丝牌） | `#FFD700` (金色) 显示牌名和等级 |
 | SC | `#0000FF` (蓝色) |
 | 礼物/上舰 | `#FF0000` (红色) |

 ### 不要

 - 不要引入新的 UI 框架或大型依赖（除非核心需求）
 - 不要在 theme handler 里硬编码颜色值 — 从 `config.Config` 读取
 - 不要阻塞 `DanmuHandler` 或 `RoomInfoHandler` — 它们跑在 channel 读取循环里
 - 不要移除 `select/default` 保护 — channel 关闭时要有安全路径

 ## 测试

 项目目前没有测试基础设施。需要添加测试时：

 - `config/` — 解析/保存逻辑可写表驱动测试
 - `login/` — Cookie 解析、验证逻辑可单元测试
 - `getter/` — HTTP 请求层可 mock，消息构建函数可单元测试
 - `sender/` — Cookie 提取、弹幕分段逻辑可单元测试
 - UI 层（tview）依赖终端，不适合常规单元测试

 ```bash
 # 运行所有测试
 go test ./...

 # 带覆盖率
 go test -coverprofile=coverage.out ./...
 go tool cover -html=coverage.out
 ```

 ## 安全注意事项

 - Cookie 存储在 `~/.config/bili/config.toml` 中，包含 `bili_jct`（CSRF token）
 - `.gitignore` 中已排除 `**/config.toml`，不要移除这条规则
 - 不要将包含真实 Cookie 的配置文件提交到 git
 - HTTP 客户端设置合理的 Timeout，避免接口挂死
 - 所有 B 站 API 调用使用统一的 User-Agent 头
 - 扫码登录使用 HTTPS，不降级到 HTTP
 - `notify-send` 命令忽略执行错误（可能在无桌面环境机器上运行）

 ## 相关技能

 开发本项目时可加载以下 ECC 系统技能：

 - `go-reviewer` — Go 代码审查
 - `go-build-resolver` — Go 编译错误排查
 - `security-review` — 安全检查（注意 Cookie 和 CSRF token 处理）
 - `coding-standards` — 通用编码规范

 ## ECC Agent 映射

 | 场景 | Agent |
 |------|-------|
 | 新增功能或重构 | planner |
 | 代码审查 | go-reviewer |
 | 编译错误 | go-build-resolver |
 | 安全审查 | security-reviewer |
 | 修改代码后审查 | code-reviewer |
 | 架构决策 | architect |
 | Bug 修复 | tdd-guide |
