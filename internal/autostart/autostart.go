// Package autostart manages the user-level launch-at-login entry.
package autostart

import "errors"

// ErrUnsupported indicates that launch at login is unavailable on this platform.
var ErrUnsupported = errors.New("当前平台不支持开机自启")
