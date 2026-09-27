package main

import (
	"fmt"
	"os"
	"time"
)

const (
	maxGroupSubscribes   = 32
	maxHeartbeatMsgCount = 3
)

type PeerInfo struct {
	IP       string
	LastSeen time.Time
}

func stateManager(events <-chan PeerEvent, sigChan <-chan os.Signal) {
	peers := make(map[uint64]PeerInfo)
	printTicker := time.NewTicker(1 * time.Second)
	defer printTicker.Stop()

	for {
		select {
		case ev := <-events:
			peers[ev.ID] = PeerInfo{IP: ev.IP, LastSeen: time.Now()}
		case <-printTicker.C:
			for id, info := range peers {
				if time.Since(info.LastSeen) > maxHeartbeatMsgCount*time.Second {
					delete(peers, id)
				}
			}
			fmt.Printf("Found live copies: %d:\n", len(peers))
			for id, info := range peers {
				fmt.Printf("\t%s - %d\n", info.IP, id)
			}
		case <-sigChan:
			fmt.Println("Terminated...")
			return
		}
	}
}
