package control

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/muxable/rtpmagic/api"
)

func ListVideoDevices() ([]*api.VideoInputDevice, error) {
	path := "/sys/class/video4linux"
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	devices := make([]*api.VideoInputDevice, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			name, err := os.ReadFile(filepath.Join(path, f.Name(), "name"))
			if err != nil {
				return nil, err
			}
			devices = append(devices, &api.VideoInputDevice{
				Id: f.Name(),
				Name: string(name),
			})
		}
	}
	return devices, nil
}
