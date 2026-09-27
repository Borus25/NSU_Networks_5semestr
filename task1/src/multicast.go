package main

import (
	"fmt"
	"net"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const multicastHopLimit = 1

func listenGroup(network string, ifi *net.Interface, groupAddr *net.UDPAddr) (*net.UDPConn, error) {
	return net.ListenMulticastUDP(network, ifi, groupAddr)
}

func configureInterfaceAndLoopback(conn *net.UDPConn, network string, ifi *net.Interface) error {
	if network == "udp4" {
		pc := ipv4.NewPacketConn(conn)
		if err := pc.SetMulticastInterface(ifi); err != nil {
			return fmt.Errorf("select IPv4 send interface: %w", err)
		}
		if err := pc.SetMulticastTTL(multicastHopLimit); err != nil {
			return fmt.Errorf("set IPv4 multicast TTL: %w", err)
		}
		return pc.SetMulticastLoopback(true)
	}

	pc := ipv6.NewPacketConn(conn)
	if err := pc.SetMulticastInterface(ifi); err != nil {
		return fmt.Errorf("select IPv6 send interface: %w", err)
	}
	if err := pc.SetMulticastHopLimit(multicastHopLimit); err != nil {
		return fmt.Errorf("set IPv6 multicast Hop Limit: %w", err)
	}
	return pc.SetMulticastLoopback(true)
}
