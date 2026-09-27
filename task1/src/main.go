package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s \n", os.Args[0])
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
		Port: groupPort,
	}

	conn, err := listenGroup(network, &groupAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListenMulticastUDP: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err := enableLoopback(conn, network); err != nil {
		fmt.Fprintf(os.Stderr, "failed to enable multicast loopback: %v\n", err)
		os.Exit(1)
	}

	peerEvents := make(chan PeerEvent, maxGroupSubscribes)

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
