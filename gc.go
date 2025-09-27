package main

import "time"

func runGC() {
	for sessionID, buf := range bufMap {
		if time.Since(buf.createTime) > 24*time.Hour {
			delete(bufMap, sessionID)
		}
	}
}

func startGC() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		runGC()
	}
}
