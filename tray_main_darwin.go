//go:build darwin

package main

/*
#include <dispatch/dispatch.h>

extern void npcRunOnMain(void);

static void npcDispatchMain(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		npcRunOnMain();
	});
}
*/
import "C"

import "sync"

var (
	mainTaskMu sync.Mutex
	mainTask   func()
)

//export npcRunOnMain
func npcRunOnMain() {
	mainTaskMu.Lock()
	task := mainTask
	mainTask = nil
	mainTaskMu.Unlock()
	if task != nil {
		task()
	}
}

func runOnMain(fn func()) {
	done := make(chan struct{})
	mainTaskMu.Lock()
	mainTask = func() {
		fn()
		close(done)
	}
	mainTaskMu.Unlock()
	C.npcDispatchMain()
	<-done
}
