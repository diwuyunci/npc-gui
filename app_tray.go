package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"npc-gui/internal/autostart"
	"npc-gui/internal/buildinfo"
	"npc-gui/internal/model"
	"npc-gui/internal/tray"
)

// SetStartHidden records whether the application was launched by autostart.
func (a *App) SetStartHidden(hidden bool) {
	a.mu.Lock()
	a.startHidden = hidden
	a.mu.Unlock()
}

func (a *App) syncAutostart(settings model.Settings) {
	if settings.LaunchAtLogin {
		if err := autostart.SetEnabled(true); err != nil {
			a.logf("autostart enable failed: %v", err)
		}
		return
	}
	if autostart.IsEnabled() {
		if err := autostart.SetEnabled(false); err != nil {
			a.logf("autostart disable failed: %v", err)
		}
	}
}

func (a *App) startTray() {
	a.mu.Lock()
	if a.tray != nil || a.quitting {
		a.mu.Unlock()
		return
	}
	t := tray.New(a)
	a.tray = t
	a.mu.Unlock()
	runOnMain(func() { t.Start() })
	a.logf("tray attached")
	go func() {
		time.Sleep(2 * time.Second)
		a.logf("tray ready=%v", t.Ready())
	}()
}

func (a *App) stopTray() {
	a.mu.Lock()
	t := a.tray
	a.mu.Unlock()
	if t != nil {
		t.Stop()
	}
}

func (a *App) trayActive() bool {
	a.mu.RLock()
	t := a.tray
	a.mu.RUnlock()
	return t != nil && t.Ready()
}

func (a *App) updateTray() {
	a.mu.RLock()
	t := a.tray
	a.mu.RUnlock()
	if t != nil {
		t.Update()
	}
}

// TraySnapshot implements tray.Controller.
func (a *App) TraySnapshot() tray.Snapshot {
	snapshot := tray.Snapshot{Version: buildinfo.Version}
	settings := a.currentSettings()
	snapshot.LaunchAtLogin = settings.LaunchAtLogin
	if a.store != nil {
		snapshot.Total = len(a.store.Profiles())
	}
	if a.manager != nil {
		for _, info := range a.manager.List() {
			if info.State == model.StateOnline {
				snapshot.Online++
			}
			if info.PID > 0 {
				snapshot.Running++
			}
		}
	}
	return snapshot
}

// ShowWindow implements tray.Controller.
func (a *App) ShowWindow() {
	ctx := a.baseContext()
	wailsRuntime.WindowUnminimise(ctx)
	wailsRuntime.WindowShow(ctx)
}

// TrayStartAll implements tray.Controller.
func (a *App) TrayStartAll() error {
	return operationErrors(a.StartAll())
}

// TrayStopAll implements tray.Controller.
func (a *App) TrayStopAll() error {
	return operationErrors(a.StopAll())
}

// TrayToggleAutostart implements tray.Controller.
func (a *App) TrayToggleAutostart() error {
	return a.toggleAutostart()
}

// TrayQuit implements tray.Controller.
func (a *App) TrayQuit() {
	go a.quitFromTray()
}

func operationErrors(results []model.OperationResult) error {
	var messages []string
	for _, result := range results {
		if !result.OK {
			message := result.Message
			if message == "" {
				message = result.Code
			}
			if message != "" {
				messages = append(messages, message)
			}
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return errors.New(strings.Join(messages, "；"))
}

func (a *App) toggleAutostart() error {
	settings := a.currentSettings()
	desired := !settings.LaunchAtLogin
	if err := autostart.SetEnabled(desired); err != nil {
		return fmt.Errorf("设置开机自启失败：%w", err)
	}
	settings.LaunchAtLogin = desired
	if a.store != nil {
		if _, err := a.store.SaveSettings(settings); err != nil {
			return fmt.Errorf("保存设置失败：%w", err)
		}
	}
	a.emit("settings:changed", settings)
	a.updateTray()
	return nil
}

func (a *App) quitFromTray() {
	a.mu.Lock()
	if a.quitting {
		a.mu.Unlock()
		return
	}
	a.quitting = true
	a.mu.Unlock()
	if a.manager != nil {
		a.manager.Close()
	}
	wailsRuntime.Quit(a.baseContext())
}

func (a *App) logf(format string, args ...any) {
	if a.logger != nil {
		a.logger.Info(fmt.Sprintf(format, args...))
	}
}
