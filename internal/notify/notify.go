// Package notify sends native desktop notifications.
package notify

import "github.com/gen2brain/beeep"

// Send displays a desktop notification.
func Send(title, message string) error {
	if title == "" {
		title = "NPC 客户端"
	}
	return beeep.Notify(title, message, "")
}
