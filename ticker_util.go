package main

import "time"

// sleepTick is a tiny shared helper so every background watcher in this
// project (voice notes, notifications, sound alerts, stats) polls on the
// same cadence without each needing its own literal duration constant.
func sleepTick() {
	time.Sleep(1 * time.Second)
}