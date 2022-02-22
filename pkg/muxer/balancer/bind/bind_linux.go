package bind

import (
	"net"
	"reflect"
	"syscall"
)

func BindToDevice(conn *net.UDPConn, device string) error {
	ptrVal := reflect.ValueOf(*conn)
	fdmember := reflect.Indirect(ptrVal).FieldByName("fd")
	pfdmember := reflect.Indirect(fdmember).FieldByName("pfd")
	netfdmember := reflect.Indirect(pfdmember).FieldByName("Sysfd")
	fd := int(netfdmember.Int())
	return syscall.SetsockoptString(fd, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, device)
}
