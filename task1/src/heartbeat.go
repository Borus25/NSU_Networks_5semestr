package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const maxMsgSize = 8

type PeerEvent struct {
	IP string
	ID uint64
}

// При ошибке отправки возвращаем её основному потоку, а не завершаем процесс.
func senderLoop(dest *net.UDPAddr, conn *net.UDPConn, myID uint64, sendErr chan<- error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	payload := make([]byte, maxMsgSize)
	binary.BigEndian.PutUint64(payload, myID)
	for range ticker.C {
		if _, err := conn.WriteToUDP(payload, dest); err != nil {
			// Канал буферизован на один элемент: отправитель не блокируется.
			sendErr <- fmt.Errorf("WriteToUDP: %w", err)
			return
		}
	}
}

func receiverLoop(conn *net.UDPConn, myID uint64, out chan<- PeerEvent) {
	buf := make([]byte, maxMsgSize)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			return // Закрытие сокета разблокирует ReadFromUDP.
		}
		if n != maxMsgSize {
			continue
		}
		remoteID := binary.BigEndian.Uint64(buf)
		if remoteID == myID {
			continue
		}
		out <- PeerEvent{IP: src.IP.String(), ID: remoteID}
	}
}
