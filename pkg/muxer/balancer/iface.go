package balancer

import (
	"net"
	"os"
	"strings"
	"syscall"
)

func GetLocalAddresses() (map[string]*net.UDPAddr, error) {
	names := make(map[string]*net.UDPAddr)
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
		if strings.HasPrefix(i.Name, "usb") || strings.HasPrefix(i.Name, "wlan") {
			var laddr *net.UDPAddr
			for _, a := range addrs {
				switch v := a.(type) {
				case *net.IPNet:
					// ignore apparently link local addresses, these might be due to AP mode.
					if v.IP.To4() != nil && !strings.HasPrefix(v.IP.String(), "10.42.") {
						laddr = &net.UDPAddr{IP: v.IP, Port: 0}
					}
				}
			}
			if laddr != nil {
				names[i.Name] = laddr
			}
		}
	}
	return names, nil
}

func DialVia(to *net.UDPAddr, via string) (net.Conn, error) {
	sfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return nil, err
	}
	if err := syscall.BindToDevice(sfd, via); err != nil {
		return nil, err
	}
	sa := &syscall.SockaddrInet4{Port: to.Port}
	copy(sa.Addr[:], to.IP.To4())
	if err := syscall.Connect(sfd, sa); err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(sfd), via)
	conn, err := net.FileConn(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	return conn, nil
}
