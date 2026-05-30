package device

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/syscmd"
)

type Dependency struct {
	Name      string `json:"name"`
	Found     bool   `json:"found"`
	Path      string `json:"path,omitempty"`
	Install   string `json:"install,omitempty"`
	Essential bool   `json:"essential"`
}

type Info struct {
	UDID        string `json:"udid,omitempty"`
	Name        string `json:"name,omitempty"`
	ProductType string `json:"product_type,omitempty"`
	IOSVersion  string `json:"ios_version,omitempty"`
	Trusted     bool   `json:"trusted"`
}

type DoctorReport struct {
	Dependencies []Dependency `json:"dependencies"`
	Device       Info         `json:"device"`
}

func Check(ctx context.Context) DoctorReport {
	deps := []Dependency{
		checkCommand("idevice_id", "brew install libimobiledevice", true),
		checkCommand("ideviceinfo", "brew install libimobiledevice", true),
		checkCommand("idevicepair", "brew install libimobiledevice", false),
		checkCommand("gphoto2", "brew install gphoto2", false),
		checkCommand("ifuse", "optional on Linux; macOS prefers Image Capture", false),
	}
	return DoctorReport{Dependencies: deps, Device: detectDevice(ctx)}
}

func checkCommand(name, install string, essential bool) Dependency {
	path, err := syscmd.LookPath(name)
	return Dependency{Name: name, Found: err == nil, Path: path, Install: install, Essential: essential}
}

func detectDevice(ctx context.Context) Info {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ideviceID, err := syscmd.LookPath("idevice_id")
	if err != nil {
		return Info{}
	}
	udidOut, err := exec.CommandContext(ctx, ideviceID, "-l").Output()
	if err != nil {
		return Info{}
	}
	udids := strings.Fields(string(udidOut))
	if len(udids) == 0 {
		return Info{}
	}

	info := Info{UDID: udids[0], Trusted: true}
	fields := map[string]*string{
		"DeviceName":     &info.Name,
		"ProductType":    &info.ProductType,
		"ProductVersion": &info.IOSVersion,
	}
	for key, target := range fields {
		value, err := ideviceInfoValue(ctx, info.UDID, key)
		if err == nil {
			*target = value
		}
	}
	return info
}

func ideviceInfoValue(ctx context.Context, udid, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ideviceInfo, err := syscmd.LookPath("ideviceinfo")
	if err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, ideviceInfo, "-u", udid, "-k", key).Output()
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(out))
	if value == "" {
		return "", errors.New("empty device info")
	}
	return value, nil
}
