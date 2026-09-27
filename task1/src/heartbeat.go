package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"
)

const maxMsgSize = 8

type PeerEvent struct {
	IP string
	ID uint64
}

func senderLoop(dest *net.UDPAddr, conn *net.UDPConn, myId uint64) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	payload := make([]byte, 8)
	binary.BigEndian.PutUint64(payload, myId)

	for range ticker.C {
		_, err := conn.WriteToUDP(payload, dest)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WriteToUDP: %v\n", err)
			os.Exit(1)
		}
	}
}

func receiverLoop(conn *net.UDPConn, myID uint64, out chan<- PeerEvent) {
	buf := make([]byte, maxMsgSize)

	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		if n != 8 {
			continue
		}
		remoteID := binary.BigEndian.Uint64(buf)
		if remoteID == myID {
			continue
		}
		out <- PeerEvent{IP: src.IP.String(), ID: remoteID}
	}
}
