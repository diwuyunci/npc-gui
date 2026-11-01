package main

import (
	"fmt"
	"strings"
	"sync"

	"npc-gui/internal/model"
	"npc-gui/internal/notify"
)

type notifier struct {
	mu         sync.Mutex
	lastStates map[string]string
	lastUpdate string
}

func (a *App) notifyEvent(event string, payload any) {
	if a.notifier == nil || !a.currentSettings().Notifications || a.isQuitting() {
		return
	}
	switch event {
	case "runtime:state":
		info, ok := payload.(model.RuntimeInfo)
		if !ok {
			return
		}
		a.notifier.mu.Lock()
		if a.notifier.lastStates == nil {
			a.notifier.lastStates = map[string]string{}
		}
		previous := a.notifier.lastStates[info.ProfileID]
		a.notifier.lastStates[info.ProfileID] = info.State
		a.notifier.mu.Unlock()

		name := a.profileName(info.ProfileID)
		switch {
		case info.State == model.StateError && previous != model.StateError:
			message := strings.TrimSpace(info.LastError)
			if message == "" {
				message = "npc 连接出现异常"
			}
			sendNotification("连接异常", name+"："+truncate(message, 140))
		case info.State == model.StateOnline && (previous == model.StateReconnecting || previous == model.StateError):
			sendNotification("连接已恢复", name+" 已重新连接到服务端")
		}
	case "update:state":
		state, ok := payload.(model.UpdateStateInfo)
		if !ok || state.State != model.UpdateAvailable || state.Latest == nil {
			return
		}
		version := state.Latest.Version
		if version == "" {
			return
		}
		a.notifier.mu.Lock()
		duplicate := a.notifier.lastUpdate == version
		a.notifier.lastUpdate = version
		a.notifier.mu.Unlock()
		if !duplicate {
			sendNotification("发现客户端新版本", fmt.Sprintf("NPC 客户端 %s 可以更新", version))
		}
	}
}

func sendNotification(title, message string) {
	if message == "" {
		return
	}
	go func() { _ = notify.Send(title, message) }()
}

func truncate(s string, limit int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}

func (a *App) profileName(id string) string {
	if a.store != nil {
		if profile, ok := a.store.GetProfile(id); ok {
			return profile.DisplayName()
		}
	}
	if id == "" {
		return "连接"
	}
	return id
}
