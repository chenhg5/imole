// Package device provides iPhone device detection and dependency checking.
package device

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strconv"
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
	UDID        string   `json:"udid,omitempty"`
	Name        string   `json:"name,omitempty"`
	ProductType string   `json:"product_type,omitempty"`
	IOSVersion  string   `json:"ios_version,omitempty"`
	Trusted     bool     `json:"trusted"`
	Connected   bool     `json:"connected"`
	Storage     *Storage `json:"storage,omitempty"`
}

type Storage struct {
	TotalDataCapacity   int64   `json:"total_data_capacity"`
	AmountDataAvailable int64   `json:"amount_data_available"`
	UsedData            int64   `json:"used_data"`
	FreePercent         float64 `json:"free_percent"`
	UsedPercent         float64 `json:"used_percent"`
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
	idCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ideviceID, err := syscmd.LookPath("idevice_id")
	if err != nil {
		return Info{}
	}
	udidOut, err := exec.CommandContext(idCtx, ideviceID, "-l").Output()
	if err != nil {
		return Info{}
	}
	udids := strings.Fields(string(udidOut))
	if len(udids) == 0 {
		return Info{}
	}

	info := Info{UDID: udids[0], Connected: true}
	fields := map[string]*string{
		"DeviceName":     &info.Name,
		"ProductType":    &info.ProductType,
		"ProductVersion": &info.IOSVersion,
	}
	trusted := false
	for key, target := range fields {
		value, err := ideviceInfoValue(ctx, info.UDID, key)
		if err == nil {
			trusted = true
			*target = value
		}
	}
	info.Trusted = trusted
	if storage, err := diskUsage(ctx, info.UDID); err == nil {
		info.Storage = &storage
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

func diskUsage(ctx context.Context, udid string) (Storage, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ideviceInfo, err := syscmd.LookPath("ideviceinfo")
	if err != nil {
		return Storage{}, err
	}
	out, err := exec.CommandContext(ctx, ideviceInfo, "-u", udid, "-q", "com.apple.disk_usage").Output()
	if err != nil {
		return Storage{}, err
	}
	return parseDiskUsage(string(out))
}

func parseDiskUsage(out string) (Storage, error) {
	values := make(map[string]int64)
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			continue
		}
		values[strings.TrimSpace(key)] = n
	}

	total := firstPositive(values["TotalDataCapacity"], values["TotalDiskCapacity"])
	available := firstPositive(values["AmountDataAvailable"], values["TotalDataAvailable"])
	if total <= 0 {
		return Storage{}, errors.New("missing TotalDataCapacity")
	}
	if available < 0 {
		available = 0
	}
	if available > total {
		available = total
	}
	used := total - available
	return Storage{
		TotalDataCapacity:   total,
		AmountDataAvailable: available,
		UsedData:            used,
		FreePercent:         float64(available) * 100 / float64(total),
		UsedPercent:         float64(used) * 100 / float64(total),
	}, nil
}

func firstPositive(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
