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

	"github.com/chenhg5/imole/internal/geo"
	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/syscmd"
)

// ── base scan ────────────────────────────────────────────────────────────────

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
		return media.Result{}, deviceNotFoundError(payload.Device)
	}

	result := imageCapturePayloadToResult(payload, opts)
	_ = writeImageCaptureCache(payload)
	return result, nil
}

// ── metadata scan ─────────────────────────────────────────────────────────────

// ScanImageCaptureWithMeta fetches full EXIF metadata (GPS, date, dimensions)
// from ImageCaptureCore without downloading the full files. Results are cached
// for 7 days since EXIF data on the device never changes.
func ScanImageCaptureWithMeta(ctx context.Context, opts media.Options) (media.Result, error) {
	// Try 7-day metadata cache first.
	if cached, ok := readImageCaptureMetaCache(opts); ok {
		return cached, nil
	}

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

	// Allow up to 90 seconds — metadata for 1000+ files can take ~30-60s.
	runCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, swift, tmp.Name())
	cmd.Env = append(os.Environ(), "IMOLE_FETCH_META=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return media.Result{}, fmt.Errorf("ImageCaptureCore metadata helper failed: %s", string(out))
	}

	var payload imageCapturePayloadMeta
	if err := json.Unmarshal(out, &payload); err != nil {
		return media.Result{}, fmt.Errorf("ImageCaptureCore metadata helper returned invalid JSON: %w: %s", err, string(out))
	}
	if len(payload.Files) == 0 {
		return media.Result{}, deviceNotFoundError(payload.Device)
	}

	result := imageCaptureMetaPayloadToResult(payload, opts)
	_ = writeImageCaptureMetaCache(payload)
	return result, nil
}

func deviceNotFoundError(device string) error {
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "" {
		return fmt.Errorf(
			"imole must run in the Mac's local Terminal app, not over SSH\n" +
				"  ImageCaptureCore cannot access the iPhone over SSH sessions")
	}
	if device == "" {
		return fmt.Errorf(
			"no iPhone detected — check the following:\n" +
				"  1. iPhone is connected via USB and the cable is data-capable (not charge-only)\n" +
				"  2. iPhone is unlocked (show the home/lock screen)\n" +
				"  3. Tap 'Trust This Computer' on the iPhone if prompted\n" +
				"  4. Try unplugging and replugging the cable")
	}
	return fmt.Errorf(
		"iPhone '%s' connected but Photos library is not accessible\n"+
			"  Most likely causes:\n"+
			"  • iCloud Photos is set to 'Optimize iPhone Storage'\n"+
			"    Fix: iPhone → Settings → Photos → Download and Keep Originals\n"+
			"  • iPhone screen is locked — unlock it and try again\n"+
			"  • Photos app privacy denied — iPhone → Settings → Privacy & Security → Photos",
		device)
}

// ── result builders ──────────────────────────────────────────────────────────

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

const exifDateFormat = "2006:01:02 15:04:05"

func imageCaptureMetaPayloadToResult(payload imageCapturePayloadMeta, opts media.Options) media.Result {
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

		// Apply metadata.
		if file.HasGPS {
			item.GPSLat = file.GPSLat
			item.GPSLon = file.GPSLon
			item.HasGPS = true
			// Offline reverse geocoding.
			loc := geo.FromGPS(file.GPSLat, file.GPSLon)
			item.Country = loc.Country
			item.CountryCode = loc.CountryCode
			item.Region = loc.Region
			item.Continent = loc.Continent
		}
		if file.TakenAt != "" {
			if t, err := time.ParseInLocation(exifDateFormat, file.TakenAt, time.Local); err == nil {
				item.TakenAt = t
			}
		}
		item.Width = file.Width
		item.Height = file.Height

		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})
	return media.Result{Summary: summarize("imagecapture:"+payload.Device, items, opts), Items: items}
}

// ── caches ───────────────────────────────────────────────────────────────────

func imageCaptureCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "imole", "imagecapture-scan.json"), nil
}

func imageCaptureMetaCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "imole", "imagecapture-meta-scan.json"), nil
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

func writeImageCaptureMetaCache(payload imageCapturePayloadMeta) error {
	path, err := imageCaptureMetaCachePath()
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

func readImageCaptureMetaCache(opts media.Options) (media.Result, bool) {
	path, err := imageCaptureMetaCachePath()
	if err != nil {
		return media.Result{}, false
	}
	info, err := os.Stat(path)
	// Metadata cache TTL: 7 days (EXIF on device never changes).
	if err != nil || time.Since(info.ModTime()) > 7*24*time.Hour {
		return media.Result{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return media.Result{}, false
	}
	var payload imageCapturePayloadMeta
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.Files) == 0 {
		return media.Result{}, false
	}
	return imageCaptureMetaPayloadToResult(payload, opts), true
}

// ── data types ───────────────────────────────────────────────────────────────

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

type imageCapturePayloadMeta struct {
	Device string                 `json:"device"`
	Files  []imageCaptureFileMeta `json:"files"`
}

type imageCaptureFileMeta struct {
	imageCaptureFile
	GPSLat  float64 `json:"gps_lat,omitempty"`
	GPSLon  float64 `json:"gps_lon,omitempty"`
	HasGPS  bool    `json:"has_gps,omitempty"`
	TakenAt string  `json:"taken_at,omitempty"` // "2006:01:02 15:04:05"
	Width   int     `json:"width,omitempty"`
	Height  int     `json:"height,omitempty"`
}

// ── Swift helper ──────────────────────────────────────────────────────────────

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
    var gps_lat: Double?
    var gps_lon: Double?
    var has_gps: Bool
    var taken_at: String?
    var width: Int?
    var height: Int?
}

final class Scanner: NSObject, ICDeviceBrowserDelegate, ICDeviceDelegate, ICCameraDeviceDelegate {
    let browser = ICDeviceBrowser()
    let isoFmt = ISO8601DateFormatter()
    var done = false
    var deviceName = ""
    var files: [MediaFile] = []
    var cameraFileRefs: [ICCameraFile] = []
    var pendingMeta = 0
    let withMeta: Bool

    override init() {
        withMeta = ProcessInfo.processInfo.environment["IMOLE_FETCH_META"] == "1"
        super.init()
        isoFmt.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        browser.delegate = self
        browser.browsedDeviceTypeMask = ICDeviceTypeMask.camera
    }

    func run() {
        browser.start()
        let timeout: TimeInterval = withMeta ? 80 : 15
        let until = Date().addingTimeInterval(timeout)
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

        if withMeta && !cameraFileRefs.isEmpty {
            pendingMeta = cameraFileRefs.count
            for file in cameraFileRefs {
                device.requestMetadata(for: file)
            }
        } else {
            done = true
        }
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
                    created_at: file.creationDate.map { isoFmt.string(from: $0) },
                    modified_at: file.modificationDate.map { isoFmt.string(from: $0) },
                    gps_lat: nil,
                    gps_lon: nil,
                    has_gps: false,
                    taken_at: nil,
                    width: nil,
                    height: nil
                ))
                if withMeta {
                    cameraFileRefs.append(file)
                }
            }
        }
    }

    func cameraDevice(_ camera: ICCameraDevice, didReceiveMetadata metadata: [AnyHashable : Any]?, for item: ICCameraItem, error: Error?) {
        guard withMeta, let file = item as? ICCameraFile else { return }

        // Match by reference identity.
        var matchIdx = -1
        for (i, ref) in cameraFileRefs.enumerated() {
            if ref === file { matchIdx = i; break }
        }

        if matchIdx >= 0, let meta = metadata {
            applyMeta(idx: matchIdx, meta: meta)
        }

        pendingMeta -= 1
        if pendingMeta <= 0 { done = true }
    }

    func applyMeta(idx: Int, meta: [AnyHashable: Any]) {
        // GPS — keys match CGImagePropertyGPS* string values.
        if let gps = meta["{GPS}"] as? [String: Any] {
            var lat = (gps["Latitude"] as? Double) ?? 0
            var lon = (gps["Longitude"] as? Double) ?? 0
            if let ref = gps["LatitudeRef"] as? String, ref == "S" { lat = -lat }
            if let ref = gps["LongitudeRef"] as? String, ref == "W" { lon = -lon }
            if lat != 0 || lon != 0 {
                files[idx].gps_lat = lat
                files[idx].gps_lon = lon
                files[idx].has_gps = true
            }
        }
        // EXIF — DateTimeOriginal, pixel dimensions.
        if let exif = meta["{Exif}"] as? [String: Any] {
            files[idx].taken_at = exif["DateTimeOriginal"] as? String
            if let w = exif["PixelXDimension"] as? Int { files[idx].width = w }
            else if let w = exif["PixelXDimension"] as? Double { files[idx].width = Int(w) }
            if let h = exif["PixelYDimension"] as? Int { files[idx].height = h }
            else if let h = exif["PixelYDimension"] as? Double { files[idx].height = Int(h) }
        }
    }

    func cameraDevice(_ camera: ICCameraDevice, didAdd items: [ICCameraItem]) {}
    func cameraDevice(_ camera: ICCameraDevice, didRemove items: [ICCameraItem]) {}
    func cameraDeviceDidChangeCapability(_ camera: ICCameraDevice) {}
    func cameraDevice(_ camera: ICCameraDevice, didReceivePTPEvent eventData: Data) {}
    func cameraDevice(_ camera: ICCameraDevice, didRenameItems items: [ICCameraItem]) {}
    func cameraDevice(_ camera: ICCameraDevice, didReceiveThumbnail thumbnail: CGImage?, for item: ICCameraItem, error: Error?) {}
    func cameraDeviceDidRemoveAccessRestriction(_ device: ICDevice) {}
    func cameraDeviceDidEnableAccessRestriction(_ device: ICDevice) {}
}

Scanner().run()
`
