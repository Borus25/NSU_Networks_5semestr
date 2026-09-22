package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type PeerEvent struct {
	IP string
	ID uint64
}

type PeerInfo struct {
	IP       string
	LastSeen time.Time
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
	buf := make([]byte, 8)

	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "recvfrom failed: %v\n", err)
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

// stateManager хранит таблицу живых копий, где КЛЮЧ = instance_id,
// а не IP-адрес - иначе несколько удалённых процессов с одинаковым
// исходящим IP "склеиваются" в одну запись.
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
				if time.Since(info.LastSeen) > 3*time.Second {
					delete(peers, id)
				}
			}

			// печать в стиле C-версии: каждую секунду, независимо от изменений
			fmt.Printf("найдено копий: %d:\n", len(peers))
			for id, info := range peers {
				fmt.Printf("\t%s - %d\n", info.IP, id)
			}

		case <-sigChan:
			fmt.Println("Завершение работы")
			return
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <listen-mcast_address>\n", os.Args[0])
		os.Exit(1)
	}

	mcastAddr := net.ParseIP(os.Args[1])

	if mcastAddr.IsMulticast() == false {
		fmt.Printf("%s is not a multicast address\n", os.Args[1])
		os.Exit(1)
	}

	network := "udp4"
	if mcastAddr.To4() == nil {
		network = "udp6"
	}

	groupAddr := net.UDPAddr{
		IP:   mcastAddr,
		Port: 1488,
	}

	conn, err := net.ListenMulticastUDP(network, nil, &groupAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListenMulticastUDP: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	if mcastAddr.To4() != nil {
		packetConn := ipv4.NewPacketConn(conn)
		if err := packetConn.SetMulticastLoopback(true); err != nil {
			fmt.Fprintf(os.Stderr, "failed to enable multicast loopback: %v\n", err)
			os.Exit(1)
		}
	} else {
		packetConn := ipv6.NewPacketConn(conn)
		if err := packetConn.SetMulticastLoopback(true); err != nil {
			fmt.Fprintf(os.Stderr, "failed to enable multicast loopback: %v\n", err)
			os.Exit(1)
		}
	}

	peerEvents := make(chan PeerEvent, 32)

	var myID uint64
	if err := binary.Read(rand.Reader, binary.BigEndian, &myID); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate instance id: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("my_id = %d\n", myID)

	go senderLoop(&groupAddr, conn, myID)
	go receiverLoop(conn, myID, peerEvents)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	stateManager(peerEvents, sigChan)
}
