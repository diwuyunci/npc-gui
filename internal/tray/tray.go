// Package tray integrates a system tray icon with the Wails event loop.
package tray

import (
	_ "embed"
	"fmt"
	"runtime"
	"sync"

	"fyne.io/systray"
)

//go:embed assets/trayTemplate.png
var trayTemplate []byte

//go:embed assets/trayIcon.png
var trayIcon []byte

//go:embed assets/trayIcon.ico
var trayICO []byte

// Snapshot is the small state rendered by the tray menu.
type Snapshot struct {
	Version       string
	Online        int
	Running       int
	Total         int
	LaunchAtLogin bool
}

// Controller is implemented by the main application.
type Controller interface {
	TraySnapshot() Snapshot
	ShowWindow()
	TrayStartAll() error
	TrayStopAll() error
	TrayToggleAutostart() error
	TrayQuit()
}

// Tray owns the menu icon and menu items.
type Tray struct {
	ctrl Controller

	mu        sync.Mutex
	header    *systray.MenuItem
	status    *systray.MenuItem
	start     *systray.MenuItem
	stop      *systray.MenuItem
	autostart *systray.MenuItem
	ready     bool
}

// New creates a tray controller.
func New(ctrl Controller) *Tray {
	return &Tray{ctrl: ctrl}
}

// Start attaches the icon to the current Cocoa/Windows event loop.
func (t *Tray) Start() {
	start, _ := systray.RunWithExternalLoop(t.onReady, func() {})
	start()
}

// Stop releases the tray menu. The process must already be exiting.
func (t *Tray) Stop() {
	t.mu.Lock()
	t.ready = false
	t.mu.Unlock()
}

func (t *Tray) onReady() {
	switch runtime.GOOS {
	case "darwin":
		systray.SetTemplateIcon(trayTemplate, trayIcon)
	case "windows":
		systray.SetIcon(trayICO)
	default:
		systray.SetIcon(trayIcon)
	}
	systray.SetTooltip("NPC 客户端")

	header := systray.AddMenuItem("NPC 客户端", "NPC GUI")
	header.Disable()
	status := systray.AddMenuItem("正在启动…", "运行状态")
	status.Disable()
	systray.AddSeparator()
	show := systray.AddMenuItem("显示主窗口", "显示 NPC 客户端")
	startAll := systray.AddMenuItem("启动全部连接", "启动所有连接")
	stopAll := systray.AddMenuItem("停止全部连接", "停止所有连接")
	systray.AddSeparator()
	autostart := systray.AddMenuItemCheckbox("开机自启", "登录时自动启动 NPC 客户端", false)
	systray.AddSeparator()
	quit := systray.AddMenuItem("退出 NPC 客户端", "停止全部连接并退出")

	t.mu.Lock()
	t.header = header
	t.status = status
	t.start = startAll
	t.stop = stopAll
	t.autostart = autostart
	t.ready = true
	t.mu.Unlock()

	t.Update()
	go t.watch(show, startAll, stopAll, autostart, quit)
}

func (t *Tray) watch(show, startAll, stopAll, autostart, quit *systray.MenuItem) {
	for {
		select {
		case <-show.ClickedCh:
			t.ctrl.ShowWindow()
		case <-startAll.ClickedCh:
			_ = t.ctrl.TrayStartAll()
			t.Update()
		case <-stopAll.ClickedCh:
			_ = t.ctrl.TrayStopAll()
			t.Update()
		case <-autostart.ClickedCh:
			_ = t.ctrl.TrayToggleAutostart()
			t.Update()
		case <-quit.ClickedCh:
			t.ctrl.TrayQuit()
			return
		}
	}
}

// Update refreshes menu labels and check state from the controller.
func (t *Tray) Update() {
	snapshot := t.ctrl.TraySnapshot()
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.ready {
		return
	}
	version := snapshot.Version
	if version == "" {
		version = "dev"
	}
	t.header.SetTitle("NPC 客户端 v" + version)
	t.status.SetTitle(fmt.Sprintf("在线 %d · 运行 %d · 档案 %d", snapshot.Online, snapshot.Running, snapshot.Total))
	systray.SetTooltip(fmt.Sprintf("NPC 客户端 · 在线 %d · 运行 %d", snapshot.Online, snapshot.Running))
	if snapshot.Total == 0 {
		t.start.Disable()
		t.stop.Disable()
	} else {
		t.start.Enable()
		t.stop.Enable()
	}
	if snapshot.LaunchAtLogin {
		t.autostart.Check()
	} else {
		t.autostart.Uncheck()
	}
}

// Ready reports whether the menu has been built.
func (t *Tray) Ready() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ready
}
