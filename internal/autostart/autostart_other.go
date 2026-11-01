//go:build !darwin && !windows && !linux

package autostart

// Supported reports whether launch at login is implemented.
func Supported() bool { return false }

// IsEnabled reports whether launch at login is enabled.
func IsEnabled() bool { return false }

// SetEnabled is unsupported on this platform.
func SetEnabled(bool) error { return ErrUnsupported }
