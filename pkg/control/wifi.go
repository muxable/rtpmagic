package control

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/muxable/rtpmagic/api"
	"go.uber.org/zap"
)

func SplitWithEscaping(s, separator, escape string) []string {
	s = strings.ReplaceAll(s, escape+separator, "\x00")
	tokens := strings.Split(s, separator)
	for i, token := range tokens {
		tokens[i] = strings.ReplaceAll(token, "\x00", separator)
	}
	return tokens
}

func isAccessPoint(id string) (bool, error) {
	ifaces, err := exec.Command("nmcli", "--terse", "-f", "device,uuid", "con", "show").Output()
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(ifaces), "\n")
	for _, line := range lines {
		tokens := SplitWithEscaping(line, ":", "\\")
		if len(tokens) < 2 {
			continue
		}
		if tokens[0] == id {
			// take the name and check if it's a hotspot
			details, err := exec.Command("nmcli", "--terse", "con", "show", tokens[1]).Output()
			if err != nil {
				return false, err
			}
			for _, line := range strings.Split(string(details), "\n") {
				tokens := SplitWithEscaping(line, ":", "\\")
				if len(tokens) < 2 {
					continue
				}
				if tokens[0] == "802-11-wireless.mode" {
					return tokens[1] == "ap", nil
				}
			}
		}
	}
	return false, nil
}

func NetScan() (*api.WifiState, error) {
	if stdout, err := exec.Command("nmcli", "dev", "wifi", "rescan").Output(); err != nil {
		zap.L().Warn("failed to rescan wifi", zap.Error(err), zap.String("stdout", string(stdout)))
	}

	ifaces, err := exec.Command("nmcli", "--terse", "dev", "status").Output()
	if err != nil {
		return nil, err
	}
	var results []*api.WifiState_Interface
	lines := strings.Split(string(ifaces), "\n")
	sort.Strings(lines)
	for _, line := range lines {
		tokens := SplitWithEscaping(line, ":", "\\")
		if len(tokens) < 2 {
			continue
		}
		switch tokens[1] {
		case "ethernet":
			if tokens[2] == "connected" {
				result := &api.WifiState_Interface{
					Id:   tokens[0],
					Type: api.WifiState_Interface_ETHERNET,
				}
				results = append(results, result)
			}
		case "wifi":
			result := &api.WifiState_Interface{
				Id:                       tokens[0],
				Type: api.WifiState_Interface_WIFI,
			}
			// check if it's running in hotspot mode
			// list access points
			aps, err := exec.Command("nmcli", "--terse", "-f", "bssid,ssid,signal,security,in-use", "dev", "wifi", "list", "ifname", tokens[0]).Output()
			if err != nil {
				return nil, err
			}
			for _, line := range strings.Split(string(aps), "\n") {
				tokens := SplitWithEscaping(line, ":", "\\")
				if len(tokens) < 4 {
					continue
				}
				signal, err := strconv.ParseFloat(tokens[2], 64)
				if err != nil {
					return nil, err
				}

				result.DiscoveredAccessPoints = append(result.DiscoveredAccessPoints, &api.WifiState_AccessPoint{
					Ssid:           tokens[1],
					Bssid:          tokens[0],
					SignalStrength: signal / 100,
					Security:       tokens[3],
				})
				if tokens[4] == "*" {
					// check if it's in access point mode
					isAP, err := isAccessPoint(result.Id)
					if err != nil {
						return nil, err
					}
					if isAP {
						result.Type = api.WifiState_Interface_AP
					}
					// in-use == true
					result.ConnectedAccessPointSsid = tokens[1]
				}
			}
			results = append(results, result)
		}
	}
	return &api.WifiState{Interfaces: results}, nil
}

func NetConnect(req *api.WifiConnectRequest) error {
	// delete all the existing connections for this interface
	stdout, err := exec.Command("nmcli", "--terse", "-f", "uuid,device", "con", "show").Output()
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(stdout), "\n") {
		tokens := SplitWithEscaping(line, ":", "\\")
		if len(tokens) < 2 {
			continue
		}
		if tokens[1] == req.InterfaceId {
			if err := exec.Command("nmcli", "con", "del", tokens[0]).Run(); err != nil {
				return err
			}
		}
	}

	connectionId := uuid.NewString()

	var args []string
	if req.Bssid == "" && req.ApModeSsid == "" {
		// this is a disconnection request.
		return nil
	} else if req.ApModeSsid != "" {
		// turn this interface into an access point.
		args = []string{"dev", "wifi", "hotspot", "ifname", req.InterfaceId, "ssid", req.ApModeSsid, "con-name", connectionId}
		if req.Password != "" {
			args = append(args, "password", req.Password)
		}
	} else {
		// connect to an access point.
		args = []string{"dev", "wifi", "connect", req.Bssid, "ifname", req.InterfaceId, "name", connectionId}
		if req.Password != "" {
			args = append(args, "password", req.Password)
		}
	}
	zap.L().Debug("nmcli args", zap.Strings("args", args))
	if err := exec.Command("nmcli", args...).Run(); err != nil {
		return err
	}
	// set autoconnect if desired
	if req.Autoconnect {
		if err := exec.Command("nmcli", "con", "mod", connectionId, "connection.autoconnect", "yes").Run(); err != nil {
			return err
		}
	}
	return nil
}