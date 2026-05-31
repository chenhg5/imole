// Package device provides iPhone device detection and dependency checking.
package device

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
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
		checkCommand("idevice_id", installHint("libimobiledevice"), true),
		checkCommand("ideviceinfo", installHint("libimobiledevice"), true),
		checkCommand("idevicepair", installHint("libimobiledevice"), false),
		checkCommand("ideviceinstaller", installHint("ideviceinstaller"), false),
		checkCommand("gphoto2", installHint("gphoto2"), false),
		checkCommand("ifuse", installHint("ifuse"), false),
	}
	return DoctorReport{Dependencies: deps, Device: detectDevice(ctx)}
}

// installHint returns the platform-appropriate install instruction for a package.
func installHint(pkg string) string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install " + pkg
	case "linux":
		switch pkg {
		case "libimobiledevice":
			return "sudo apt install libimobiledevice-utils  # or: sudo dnf install libimobiledevice"
		case "gphoto2":
			return "sudo apt install gphoto2  # or: sudo dnf install gphoto2"
		case "ifuse":
			return "sudo apt install ifuse  # or: sudo dnf install ifuse"
		default:
			return "sudo apt install " + pkg
		}
	case "windows":
		switch pkg {
		case "libimobiledevice":
			return "install iTunes (includes libimobiledevice drivers)"
		case "gphoto2":
			return "not available on Windows; use --source PATH after mounting via iTunes"
		case "ifuse":
			return "not available on Windows; use --source PATH after mounting via iTunes"
		default:
			return "see https://github.com/chenhg5/imole for Windows setup"
		}
	default:
		return "see https://github.com/chenhg5/imole for setup instructions"
	}
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
