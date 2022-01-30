package balancer

import (
	"log"
	"net"
	"strings"

	"github.com/muxable/rtpmagic/pkg/muxer/balancer/bind"
)

func GetLocalAddresses() (map[string]bool, error) {
	names := make(map[string]bool)
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		// check that it has a valid gateway address.
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		if strings.HasPrefix(i.Name, "usb") || strings.HasPrefix(i.Name, "eth") {
			hasIPv4 := false
			for _, a := range addrs {
				switch v := a.(type) {
				case *net.IPNet:
					if v.IP.To4() != nil {
						hasIPv4 = true
					}
				}
			}
			if hasIPv4 {
				names[i.Name] = true
			}
		}
	}
	return names, nil
}

func DialVia(to *net.UDPAddr, via string) (*net.UDPConn, error) {
	conn, err := net.DialUDP("udp", nil, to)
	if err != nil {
		log.Printf("error dialing to %v %v", to, err)
		return nil, err
	}
	if err := bind.BindToDevice(*conn, via); err != nil {
		log.Printf("error binding to device %v %v", via, err)
		return nil, err
	}
	return conn, nil
}
