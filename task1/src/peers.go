package main

import (
	"fmt"
	"os"
	"slices"
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

func printPeers(peers map[uint64]PeerInfo) {
	ids := make([]uint64, 0, len(peers))
	for id := range peers {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	fmt.Printf("Found live copies: %d:\n", len(peers))
	for _, id := range ids {
		fmt.Printf("\t%s - %d\n", peers[id].IP, id)
	}
}

func stateManager(events <-chan PeerEvent, sigChan <-chan os.Signal, sendErr <-chan error) error {
	peers := make(map[uint64]PeerInfo)
	checkTicker := time.NewTicker(time.Second)
	defer checkTicker.Stop()

	for {
		select {
		case ev := <-events:
			now := time.Now()
			old, exists := peers[ev.ID]
			peers[ev.ID] = PeerInfo{IP: ev.IP, LastSeen: now}
			// Повторный heartbeat обновляет только время, не печатая список.
			if !exists || old.IP != ev.IP {
				printPeers(peers)
			}

		case now := <-checkTicker.C:
			changed := false
			for id, info := range peers {
				if now.Sub(info.LastSeen) > maxHeartbeatMsgCount*time.Second {
					delete(peers, id)
					changed = true
				}
			}
			if changed {
				printPeers(peers)
			}

		case err := <-sendErr:
			return err
		case <-sigChan:
			fmt.Println("Terminated...")
			return nil
		}
	}
}
