package main

import (
	"net"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const groupPort = 1488

func listenGroup(network string, groupAddr *net.UDPAddr) (*net.UDPConn, error) {
	return net.ListenMulticastUDP(network, nil, groupAddr)
}

func enableLoopback(conn *net.UDPConn, network string) error {
	if network == "udp4" {
		return ipv4.NewPacketConn(conn).SetMulticastLoopback(true)
	}
	return ipv6.NewPacketConn(conn).SetMulticastLoopback(true)
}
