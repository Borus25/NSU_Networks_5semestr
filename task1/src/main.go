package main

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func run() error {
	if len(os.Args) != 4 {
		return fmt.Errorf("usage: %s MULTICAST_IP INTERFACE PORT", os.Args[0])
	}

	mcastAddr := net.ParseIP(os.Args[1])
	if mcastAddr == nil || !mcastAddr.IsMulticast() {
		return fmt.Errorf("%s is not a multicast address", os.Args[1])
	}

	ifi, err := net.InterfaceByName(os.Args[2])
	if err != nil {
		return fmt.Errorf("interface %q: %w", os.Args[2], err)
	}
	if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagMulticast == 0 {
		return fmt.Errorf("interface %q must be up and support multicast", ifi.Name)
	}

	port, err := strconv.Atoi(os.Args[3])
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid UDP port %q: expected an integer from 1 to 65535", os.Args[3])
	}

	network := "udp4"
	if mcastAddr.To4() == nil {
		network = "udp6"
	}

	groupAddr := net.UDPAddr{IP: mcastAddr, Port: port}
	if network == "udp6" {
		groupAddr.Zone = ifi.Name
	}

	conn, err := listenGroup(network, ifi, &groupAddr)
	if err != nil {
		return fmt.Errorf("ListenMulticastUDP: %w", err)
	}
	defer conn.Close()

	if err := configureInterfaceAndLoopback(conn, network, ifi); err != nil {
		return fmt.Errorf("multicast setup: %w", err)
	}

	peerEvents := make(chan PeerEvent, maxGroupSubscribes)
	var myID uint64
	if err := binary.Read(rand.Reader, binary.BigEndian, &myID); err != nil {
		return fmt.Errorf("failed to generate instance id: %w", err)
	}
	fmt.Printf("my_id = %d\n", myID)

	// Оба цикла завершаются после закрытия сокета. Основной поток ждёт их
	// завершения перед возвратом из run().
	senderDone := make(chan struct{})
	receiverDone := make(chan struct{})
	sendErr := make(chan error, 1)
	go func() {
		defer close(senderDone)
		senderLoop(&groupAddr, conn, myID, sendErr)
	}()
	go func() {
		defer close(receiverDone)
		receiverLoop(conn, myID, peerEvents)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// stateManager возвращает ошибку отправки либо nil после сигнала.
	result := stateManager(peerEvents, sigChan, sendErr)
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		fmt.Fprintf(os.Stderr, "close UDP socket: %v\n", err)
	}
	<-senderDone
	<-receiverDone
	return result
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
