# 构建资源目录

build/ 目录保存 Wails 构建所需的图标、平台清单和打包配置。

## 目录结构

- bin/ 是构建输出目录。
- darwin/ 保存 macOS 专用文件。
- windows/ 保存 Windows 专用文件。

## macOS

darwin 目录包含 Wails 构建使用的 macOS 清单文件：

- Info.plist 是 wails build 使用的主 plist 文件。
- Info.dev.plist 是 wails dev 使用的 plist 文件。

这些文件可以在项目需要时定制。自定义文件被删除后，下一次 wails build 会重新生成默认版本。

## Windows

windows 目录包含 wails build 使用的清单、图标和安装器配置：

- icon.ico 是 Windows 应用图标。该文件缺失时，构建过程会根据 build/appicon.png 生成新的 icon.ico。
- installer/ 保存 Windows 安装器使用的文件。
- info.json 保存 Windows 构建的应用信息，安装器和可执行文件属性都会读取这些信息。
- wails.exe.manifest 是 Windows 应用的主清单文件。
