//go:build !darwin

package main

func runOnMain(fn func()) { fn() }
