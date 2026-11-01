# 技术说明

[English](technical.md) | 中文

## 技术栈

| 层面 | 技术 |
|---|---|
| 桌面框架 | Wails v2.12.0 |
| 后端 | Go 1.25.x，独立 Go module |
| 前端 | Vue 3、TypeScript、Vite、Pinia、Vue Router |
| 图标 | lucide-vue-next |
| 样式 | 原生 CSS 和 design tokens |
| 包管理 | pnpm 10.x |
| 测试 | Go testing、Vitest、Vue Test Utils |
| 发布 | GitHub Actions 和 GitHub Releases |

## 上游约束与架构决策

npc 来自 [djylb/nps](https://github.com/djylb/nps)，当前使用的版本为 0.34.10，core 版本为 0.34.0。

| 上游事实 | 技术决策 |
|---|---|
| npc 支持命令行与 -config 配置文件模式 | 应用统一使用配置文件模式启动 |
| 配置文件 [common] 只有一个 server_addr | 一个档案对应一个服务端 |
| 命令行模式支持多服务端逗号列表，但会把 vkey 放入 argv | 导入命令时拆分为多个档案，启动时使用配置文件 |
| 配置解析失败时 StartFromFile 调用 os.Exit(1)，日志器和状态是全局变量 | 应用前置校验，并使用独立子进程隔离 |
| npc 日志使用 zerolog ConsoleWriter | 应用捕获 stdout，并配合 -log=stdout 和 -log_color=false 解析 |
| C SDK 只覆盖基础连接、状态和日志快照 | 项目不采用嵌入 SDK 方案 |
| 上游 go.mod 要求 Go 1.27，项目使用 Go 1.25.x | GUI 不引入 nps 模块，保持独立构建 |
| 上游发布资产命名为 os_arch_client.tar.gz | 应用按平台映射资产并校验 GitHub asset digest |
| 上游许可证为 GPLv3 | 应用不随包再分发 npc，运行时从上游获取 |

架构采用 **sidecar 子进程**方案。GUI 负责配置、进程、日志和更新，npc 继续使用官方二进制和完整配置能力。

## 总体架构

~~~text
Vue 3 UI
  ↕ Wails bindings / events
Go App facade
  ├─ Store
  ├─ ConfigGen
  ├─ BinaryManager
  ├─ ProcessManager
  ├─ LogHub
  ├─ UpdateManager
  └─ System
~~~

| 组件 | 职责 |
|---|---|
| Store | 档案、设置、原子 JSON 持久化 |
| ConfigGen | 表单校验、npc.conf 生成、CLI 参数、NPS 命令解析 |
| BinaryManager | npc 探测、上游下载、SHA-256、版本校验 |
| ProcessManager | 进程生命周期、PID 登记、停止、重连 |
| LogHub | stdout/stderr 解析、环形缓冲、批量事件 |
| UpdateManager | 客户端 Release 检查、helper 替换、回滚 |
| System | 托盘、开机自启、通知 |

后端通过 Wails bindings 暴露档案、进程、日志、设置和更新方法，并通过 Wails events 推送运行时状态、日志、更新进度和通知事件。前端使用 Pinia 管理档案、运行状态、日志和更新状态。

## 启动流程

1. 用户保存档案后，App.SaveProfile 校验档案并写入 profiles.json。
2. 应用生成 profiles/<id>/npc.conf（0600），并计算配置哈希。
3. App.StartProfile 解析 npc 路径后，ProcessManager 使用 `npc -config=... -log=stdout -debug=false -log_color=false -log_level=... -disable_remote_update=true` 启动子进程。
4. LogHub 接收 stdout 和 stderr，剥离 ANSI 转义、解析时间和等级、更新状态机，并写入环形缓冲和日志文件。
5. 状态变化通过 runtime:state 事件推送，日志按批次通过 runtime:logs 推送。
6. 用户停止连接时，Unix 版本向进程组发送 SIGTERM，超时后发送 SIGKILL；Windows 使用 Process.Kill 或 taskkill 兜底。

## 状态机与日志

~~~text
stopped -> starting -> running -> online
                 |         |
                 |         +-> reconnecting -> online
                 |         |
                 +--------> error
online/reconnecting -> stopping -> stopped
~~~

- running 表示进程存活但尚未出现桥接成功日志，适用于 P2P 和 secret 等模式。
- online 状态由 Successful connection with server 日志触发。
- reconnecting 状态由 The connection server failed、Failed to connect、Reconnecting... 和 Client closed 等日志触发。
- 配置和认证类错误由 loading error、missing [common]、web_user may have been occupied 等日志识别。
- 进程非用户主动退出时，应用保留退出码和错误摘要。

日志链路采用环形缓冲和批量推送。缓冲区默认保留 5000 行，日志控制台支持按档案、等级和关键词过滤，并提供跟随滚动、自动换行、清空和导出。日志文件权限为 0600，并按大小轮转。

## 配置、数据与安全

### 数据目录

| 平台 | 目录 |
|---|---|
| macOS | ~/Library/Application Support/npc-gui/ |
| Windows | %AppData%/npc-gui/ |
| Linux | ~/.config/npc-gui/ |

数据目录包含以下内容：

- profiles.json、settings.json 和 binary.json 保存应用配置。
- profiles/<id>/npc.conf 保存档案配置，权限为 0600。
- logs/<id>/session-*.log 保存运行日志，权限为 0600。
- updates/ 保存客户端更新暂存与日志。
- runtime/<id>.json 保存 PID 登记，用于残留进程清理。

### 配置生成

- 表单字段经过校验后生成 npc.conf。
- 应用通过 -config 参数启动 npc。
- 高级配置支持额外 common 键值和额外 INI 段落。

### 安全边界

- 所有命令统一通过 exec.Command 传递，不经过 shell。
- vkey 只写入 0600 配置文件和档案 JSON，不进入启动 argv。
- 配置预览默认打码；真实配置和含密钥档案的导出会显示提示。
- npc 下载校验 GitHub asset digest；解压拒绝路径穿越和符号链接。
- 客户端更新只从编译期注入的仓库获取，发布构建不允许用户修改更新地址。
- 日志可能包含 npc 打印的 web 访问凭据，因此导出前需要确认分享范围。

## 系统集成

- 托盘使用 fyne.io/systray 的 RunWithExternalLoop 接入 Wails 事件循环。macOS 通过 dispatch_async 在主线程创建 NSStatusItem。
- 托盘菜单读取在线数、运行数、档案数和开机自启状态，并提供显示窗口、全部启动、全部停止和退出入口。
- 开机自启按平台注册：macOS 使用 LaunchAgent，Windows 使用 HKCU Run，Linux 使用 autostart desktop entry。
- 自启命令包含 --hidden，应用以隐藏窗口启动并由托盘接管。
- 系统通知使用 beeep，在连接异常、连接恢复和发现客户端新版本时发送。
- 托盘开关在下次启动时生效，避免运行中销毁 NSStatusItem 导致 Wails 事件循环停止。

## 客户端更新

### 版本来源

- 版本号来自 git tag，去掉前缀 v 后通过 ldflags 注入 buildinfo.Version。
- 仓库标识来自 CI 环境变量 github.repository，并注入 buildinfo.Repo。
- 本地开发构建的 Version 为 0.1.0-dev，Repo 为空，更新状态为 disabled，不访问 GitHub。
- 开发环境可以使用 NPC_GUI_UPDATE_REPO 和 NPC_GUI_UPDATE_API 指向测试 release 源，发布构建会忽略 API 覆盖。

### Release 资产

| 平台 | 资产 |
|---|---|
| macOS arm64 | npc-gui_<version>_darwin_arm64.app.zip |
| macOS amd64 | npc-gui_<version>_darwin_amd64.app.zip |
| Windows amd64 | npc-gui_<version>_windows_amd64.exe |
| Windows amd64 安装器 | npc-gui_<version>_windows_amd64_installer.exe |
| Linux amd64 | npc-gui_<version>_linux_amd64.tar.gz |
| 全部 | checksums.txt |

macOS 更新完整 .app，Windows 和 Linux 更新可执行文件。Windows 便携版 .exe 继续作为自动更新资产；NSIS 安装器把应用安装到 %LOCALAPPDATA%\Programs 并写入当前用户卸载项，因此更新器无需提权即可替换可执行文件。

### 检查、下载与校验

- 客户端通过 GET https://api.github.com/repos/<repo>/releases/latest 查询最新版本。
- latest 响应会缓存到 updates/latest.json，自动检查最多每 24 小时执行一次。
- 版本比较使用 semver，latest 接口排除 draft 和 prerelease。
- 资产按 runtime.GOOS 和 runtime.GOARCH 选择，资产缺失时界面展示 Release 页面。
- 下载内容暂存到 updates/<version>/。
- SHA-256 来自 GitHub asset digest 与 checksums.txt，两者都缺失时客户端拒绝自动更新。
- zip 和 tar 解压过程限制路径穿越、符号链接和超大文件。

### 应用与回滚

1. 用户确认“重启并更新”后，应用先停止所有 npc 进程。
2. 应用使用 `helper --apply-update --target <exe/app> --staged <bin> --staged-app <app> --kind app|binary --old-pid <pid> --relaunch <path>` 启动 helper。
3. helper 等待旧 PID 退出，等待上限为 90 秒。
4. 在 macOS 上，helper 将当前 .app 重命名为 .app.old，把新 .app 复制到目标位置，失败时恢复旧版本。
5. 在 Windows 和 Linux 上，helper 将目标可执行文件重命名为 .old，写入新文件，失败时恢复旧版本。
6. helper 启动新版本，并把结果写入 updates/last-result.json，日志写入 updates/update.log。

macOS 应用位于 AppTranslocation 或只读目录时，客户端拒绝自动更新，并提示用户移动到“应用程序”目录或手动下载。目标目录不可写时，客户端提示手动下载，不尝试提权。

## 构建与发布

### 开发构建

本地开发构建使用 dev 版本号，默认不检查客户端更新。Windows 安装器构建需要 NSIS 提供的 makensis。构建命令如下：

~~~bash
./scripts/build-local.sh
~~~

构建产物包括 macOS 应用包 build/bin/npc-gui.app，以及 dist/npc-gui_<version>_darwin_arm64.app.zip。

发布构建由 CI 注入仓库标识与 tag 版本，命令示例如下：

~~~bash
VERSION=0.1.1 REPO=owner/npc-gui PLATFORM=darwin/arm64 ./scripts/build-local.sh
~~~

### 发布流程

- CI 从 github.repository 读取仓库标识，并注入 buildinfo.Repo。
- 推送 v* 标签会触发 Release 工作流。
- Release 工作流在 macOS arm64/amd64、Windows amd64 和 Linux amd64 原生 runner 上执行构建。
- 发布资产包括 .app.zip、.exe、.tar.gz 和 checksums.txt。

## 项目结构

~~~text
npc-gui/
├── app.go
├── main.go
├── internal/
│   ├── model/
│   ├── store/
│   ├── configgen/
│   ├── npcbin/
│   ├── npcrun/
│   ├── update/
│   ├── tray/
│   ├── autostart/
│   ├── notify/
│   ├── buildinfo/
│   └── platform/
├── frontend/
├── build/
├── scripts/
├── docs/
└── .github/workflows/
~~~

## 卸载与清理

- 应用没有独立卸载器，卸载前可以在设置页关闭“开机自启”，以移除对应的 LaunchAgent、Run 项或 autostart 文件。
- 应用数据位于用户数据目录；卸载后可以手动删除该目录，配置、密钥、日志和更新缓存会随之移除。
- 托盘、通知和受管 npc 进程由应用进程管理；应用退出后托盘和通知消失，残留 npc 会在下次启动时清理。
- 如果应用被直接删除且没有关闭开机自启，操作系统登录项会保留，需要手动删除对应文件或注册表项。

## 测试与验收

Go 单测覆盖 store、configgen、ghrelease、npcbin、npcrun、update、autostart 和 helper。前端检查包括 vue-tsc、Vitest 和 Vite 生产构建。发布前的手工验收由 [manual-test.zh.md](manual-test.zh.md) 负责。

## 相关文档

- [business.zh.md](business.zh.md) 记录产品定位、目标用户、业务场景和业务边界。
- [manual-test.zh.md](manual-test.zh.md) 记录发布前的手工验收清单。
- [README.zh.md](../README.zh.md) 记录项目入口、功能和快速开始。
