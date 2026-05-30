//go:build darwin

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/syscmd"
)

func ScanImageCapture(ctx context.Context, opts media.Options) (media.Result, error) {
	swift, err := syscmd.LookPath("swift")
	if err != nil {
		return media.Result{}, fmt.Errorf("swift not found for ImageCaptureCore helper")
	}

	tmp, err := os.CreateTemp("", "imole-imagecapture-*.swift")
	if err != nil {
		return media.Result{}, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(imageCaptureSwiftSource); err != nil {
		_ = tmp.Close()
		return media.Result{}, err
	}
	if err := tmp.Close(); err != nil {
		return media.Result{}, err
	}

	runCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	out, err := exec.CommandContext(runCtx, swift, tmp.Name()).CombinedOutput()
	if err != nil {
		return media.Result{}, fmt.Errorf("ImageCaptureCore helper failed: %s", string(out))
	}

	var payload imageCapturePayload
	if err := json.Unmarshal(out, &payload); err != nil {
		return media.Result{}, fmt.Errorf("ImageCaptureCore helper returned invalid JSON: %w: %s", err, string(out))
	}
	if len(payload.Files) == 0 {
		if cached, ok := readImageCaptureCache(opts); ok {
			return cached, nil
		}
		if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "" {
			return media.Result{}, fmt.Errorf("ImageCaptureCore did not expose any media files from this SSH session; run iMole from the Mac's Terminal app")
		}
		return media.Result{}, fmt.Errorf("ImageCaptureCore did not expose any media files")
	}

	result := imageCapturePayloadToResult(payload, opts)
	_ = writeImageCaptureCache(payload)
	return result, nil
}

func imageCapturePayloadToResult(payload imageCapturePayload, opts media.Options) media.Result {
	items := make([]media.Item, 0, len(payload.Files))
	for _, file := range payload.Files {
		modTime := file.ModifiedAt
		if modTime.IsZero() {
			modTime = file.CreatedAt
		}
		path := filepath.Join("imagecapture", filepath.FromSlash(file.Path))
		item := media.NewItem("imagecapture", path, file.Size, modTime)
		item.SourcePath = "imagecapture://" + file.Path
		item.RelPath = file.Path
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})
	return media.Result{Summary: summarize("imagecapture:"+payload.Device, items, opts), Items: items}
}

func imageCaptureCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "imole", "imagecapture-scan.json"), nil
}

func writeImageCaptureCache(payload imageCapturePayload) error {
	path, err := imageCaptureCachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readImageCaptureCache(opts media.Options) (media.Result, bool) {
	path, err := imageCaptureCachePath()
	if err != nil {
		return media.Result{}, false
	}
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > 30*time.Minute {
		return media.Result{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return media.Result{}, false
	}
	var payload imageCapturePayload
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.Files) == 0 {
		return media.Result{}, false
	}
	return imageCapturePayloadToResult(payload, opts), true
}

type imageCapturePayload struct {
	Device string             `json:"device"`
	Files  []imageCaptureFile `json:"files"`
}

type imageCaptureFile struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
}

const imageCaptureSwiftSource = `
import Foundation
import ImageCaptureCore

struct Output: Encodable {
    let device: String
    let files: [MediaFile]
}

struct MediaFile: Encodable {
    let path: String
    let size: Int64
    let created_at: String?
    let modified_at: String?
}

final class Scanner: NSObject, ICDeviceBrowserDelegate, ICDeviceDelegate, ICCameraDeviceDelegate {
    let browser = ICDeviceBrowser()
    let formatter = ISO8601DateFormatter()
    var done = false
    var deviceName = ""
    var files: [MediaFile] = []

    override init() {
        super.init()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        browser.delegate = self
        browser.browsedDeviceTypeMask = ICDeviceTypeMask.camera
    }

    func run() {
        browser.start()
        let until = Date().addingTimeInterval(15)
        while !done && Date() < until {
            RunLoop.current.run(mode: .default, before: Date().addingTimeInterval(0.2))
        }
        browser.stop()
        emit()
    }

    func emit() {
        let output = Output(device: deviceName, files: files)
        let encoder = JSONEncoder()
        if let data = try? encoder.encode(output), let text = String(data: data, encoding: .utf8) {
            print(text)
        } else {
            print("{\"device\":\"\",\"files\":[]}")
        }
    }

    func deviceBrowser(_ browser: ICDeviceBrowser, didAdd device: ICDevice, moreComing: Bool) {
        device.delegate = self
        if let camera = device as? ICCameraDevice {
            deviceName = camera.name ?? ""
            camera.delegate = self
            camera.requestOpenSession()
        }
    }

    func deviceBrowser(_ browser: ICDeviceBrowser, didRemove device: ICDevice, moreGoing: Bool) {}
    func didRemove(_ device: ICDevice) {}
    func device(_ device: ICDevice, didOpenSessionWithError error: Error?) {}
    func device(_ device: ICDevice, didCloseSessionWithError error: Error?) {}

    func deviceDidBecomeReady(withCompleteContentCatalog device: ICCameraDevice) {
        deviceName = device.name ?? deviceName
        collect(items: device.contents ?? [], prefix: "")
        done = true
    }

    func collect(items: [ICCameraItem], prefix: String) {
        for item in items {
            if let folder = item as? ICCameraFolder {
                let name = folder.name ?? ""
                collect(items: folder.contents ?? [], prefix: prefix + name + "/")
            } else if let file = item as? ICCameraFile {
                let name = file.name ?? ""
                let path = prefix + name
                files.append(MediaFile(
                    path: path,
                    size: Int64(file.fileSize),
                    created_at: file.creationDate.map { formatter.string(from: $0) },
                    modified_at: file.modificationDate.map { formatter.string(from: $0) }
                ))
            }
        }
    }

    func cameraDevice(_ camera: ICCameraDevice, didAdd items: [ICCameraItem]) {}
    func cameraDevice(_ camera: ICCameraDevice, didRemove items: [ICCameraItem]) {}
    func cameraDeviceDidChangeCapability(_ camera: ICCameraDevice) {}
    func cameraDevice(_ camera: ICCameraDevice, didReceivePTPEvent eventData: Data) {}
    func cameraDevice(_ camera: ICCameraDevice, didRenameItems items: [ICCameraItem]) {}
    func cameraDevice(_ camera: ICCameraDevice, didReceiveThumbnail thumbnail: CGImage?, for item: ICCameraItem, error: Error?) {}
    func cameraDevice(_ camera: ICCameraDevice, didReceiveMetadata metadata: [AnyHashable : Any]?, for item: ICCameraItem, error: Error?) {}
    func cameraDeviceDidRemoveAccessRestriction(_ device: ICDevice) {}
    func cameraDeviceDidEnableAccessRestriction(_ device: ICDevice) {}
}

Scanner().run()
`
