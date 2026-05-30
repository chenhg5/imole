//go:build darwin

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/chenhg5/imole/internal/progress"
	"github.com/chenhg5/imole/internal/syscmd"
)

type imageCaptureDownloadSelection struct {
	Items []imageCaptureDownloadItem `json:"items"`
}

type imageCaptureDownloadItem struct {
	Path    string `json:"path"`
	DestRel string `json:"dest_rel"`
	Size    int64  `json:"size"`
}

func DownloadImageCapture(ctx context.Context, requests []DownloadRequest, destRoot string) ([]DownloadResult, error) {
	if len(requests) == 0 {
		return nil, nil
	}
	swift, err := syscmd.LookPath("swift")
	if err != nil {
		return nil, fmt.Errorf("swift not found for ImageCaptureCore helper")
	}

	helper, err := os.CreateTemp("", "imole-imagecapture-download-*.swift")
	if err != nil {
		return nil, err
	}
	defer os.Remove(helper.Name())
	if _, err := helper.WriteString(imageCaptureDownloadSwiftSource); err != nil {
		_ = helper.Close()
		return nil, err
	}
	if err := helper.Close(); err != nil {
		return nil, err
	}

	selectionFile, err := os.CreateTemp("", "imole-imagecapture-selection-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(selectionFile.Name())

	selection := imageCaptureDownloadSelection{Items: make([]imageCaptureDownloadItem, 0, len(requests))}
	for _, req := range requests {
		selection.Items = append(selection.Items, imageCaptureDownloadItem{
			Path:    req.Item.RelPath,
			DestRel: req.DestRel,
			Size:    req.Item.Size,
		})
	}
	if err := json.NewEncoder(selectionFile).Encode(selection); err != nil {
		_ = selectionFile.Close()
		return nil, err
	}
	if err := selectionFile.Close(); err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(ctx, 12*time.Hour)
	defer cancel()
	cmd := exec.CommandContext(runCtx, swift, helper.Name(), selectionFile.Name(), destRoot)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.MultiWriter(progress.NewWriter(os.Stderr), &stderr)
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ImageCaptureCore download helper failed: %s", stderr.String())
	}

	var results []DownloadResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("ImageCaptureCore download helper returned invalid JSON: %w: %s", err, stdout.String())
	}
	return results, nil
}

const imageCaptureDownloadSwiftSource = `
import Foundation
import ImageCaptureCore

struct Selection: Decodable {
    let items: [SelectionItem]
}

struct SelectionItem: Decodable {
    let path: String
    let dest_rel: String
    let size: Int64
}

struct ResultItem: Encodable {
    let source_rel: String
    let dest_rel: String
    let verified: Bool
    let skipped: Bool
    let error: String?
}

final class Downloader: NSObject, ICDeviceBrowserDelegate, ICDeviceDelegate, ICCameraDeviceDelegate, ICCameraDeviceDownloadDelegate {
    let browser = ICDeviceBrowser()
    let selection: Selection
    let destRoot: URL
    var catalogReady = false
    var done = false
    var catalog: [String: ICCameraFile] = [:]
    var results: [ResultItem] = []
    var currentIndex = 0
    var currentDownloaded: Int64 = 0
    var currentItem: SelectionItem?
    var camera: ICCameraDevice?

    init(selection: Selection, destRoot: URL) {
        self.selection = selection
        self.destRoot = destRoot
        super.init()
        browser.delegate = self
        browser.browsedDeviceTypeMask = ICDeviceTypeMask.camera
    }

    func run() {
        browser.start()
        let catalogUntil = Date().addingTimeInterval(60)
        while !catalogReady && Date() < catalogUntil {
            RunLoop.current.run(mode: .default, before: Date().addingTimeInterval(0.2))
        }
        if !catalogReady {
            for item in selection.items {
                results.append(ResultItem(source_rel: item.path, dest_rel: item.dest_rel, verified: false, skipped: false, error: "timeout waiting for ImageCaptureCore catalog"))
            }
            done = true
        }
        while !done {
            RunLoop.current.run(mode: .default, before: Date().addingTimeInterval(0.2))
        }
        browser.stop()
        emit()
    }

    func emit() {
        let encoder = JSONEncoder()
        if let data = try? encoder.encode(results), let text = String(data: data, encoding: .utf8) {
            print(text)
        } else {
            print("[]")
        }
    }

    func deviceBrowser(_ browser: ICDeviceBrowser, didAdd device: ICDevice, moreComing: Bool) {
        device.delegate = self
        if let cam = device as? ICCameraDevice {
            camera = cam
            cam.delegate = self
            cam.requestOpenSession()
        }
    }

    func deviceBrowser(_ browser: ICDeviceBrowser, didRemove device: ICDevice, moreGoing: Bool) {}
    func didRemove(_ device: ICDevice) {}
    func device(_ device: ICDevice, didOpenSessionWithError error: Error?) {}
    func device(_ device: ICDevice, didCloseSessionWithError error: Error?) {}

    func deviceDidBecomeReady(withCompleteContentCatalog device: ICCameraDevice) {
        catalog.removeAll()
        collect(items: device.contents ?? [], prefix: "")
        catalogReady = true
        downloadNext()
    }

    func collect(items: [ICCameraItem], prefix: String) {
        for item in items {
            if let folder = item as? ICCameraFolder {
                let name = folder.name ?? ""
                collect(items: folder.contents ?? [], prefix: prefix + name + "/")
            } else if let file = item as? ICCameraFile {
                let name = file.name ?? ""
                catalog[prefix + name] = file
            }
        }
    }

    func downloadNext() {
        if currentIndex >= selection.items.count {
            done = true
            return
        }
        let item = selection.items[currentIndex]
        currentItem = item
        currentDownloaded = 0
        logProgress("start", item: item, downloaded: 0, total: item.size)
        let dest = destRoot.appendingPathComponent(item.dest_rel)
        let destDir = dest.deletingLastPathComponent()
        do {
            try FileManager.default.createDirectory(at: destDir, withIntermediateDirectories: true)
            if let attrs = try? FileManager.default.attributesOfItem(atPath: dest.path),
               let size = attrs[.size] as? NSNumber,
               size.int64Value == item.size {
                results.append(ResultItem(source_rel: item.path, dest_rel: item.dest_rel, verified: true, skipped: true, error: nil))
                logProgress("skip", item: item, downloaded: item.size, total: item.size)
                currentIndex += 1
                downloadNext()
                return
            }
        } catch {
            results.append(ResultItem(source_rel: item.path, dest_rel: item.dest_rel, verified: false, skipped: false, error: error.localizedDescription))
            logProgress("error", item: item, downloaded: 0, total: item.size)
            currentIndex += 1
            downloadNext()
            return
        }
        guard let file = catalog[item.path], let camera = camera else {
            results.append(ResultItem(source_rel: item.path, dest_rel: item.dest_rel, verified: false, skipped: false, error: "file not found in ImageCaptureCore catalog"))
            logProgress("error", item: item, downloaded: 0, total: item.size)
            currentIndex += 1
            downloadNext()
            return
        }
        let options: [ICDownloadOption: Any] = [
            ICDownloadOption.downloadsDirectoryURL: destDir,
            ICDownloadOption.saveAsFilename: dest.lastPathComponent
        ]
        camera.requestDownloadFile(file, options: options, downloadDelegate: self, didDownloadSelector: #selector(didDownloadFile(_:error:options:contextInfo:)), contextInfo: nil)
    }

    @objc func didDownloadFile(_ file: ICCameraFile, error: Error?, options: [String: Any], contextInfo: UnsafeMutableRawPointer?) {
        let item = currentItem
        if let item = item {
            let dest = destRoot.appendingPathComponent(item.dest_rel)
            var verified = false
            if let attrs = try? FileManager.default.attributesOfItem(atPath: dest.path),
               let size = attrs[.size] as? NSNumber {
                verified = size.int64Value == item.size
            }
            results.append(ResultItem(source_rel: item.path, dest_rel: item.dest_rel, verified: verified, skipped: false, error: error?.localizedDescription))
            logProgress(verified ? "done" : "error", item: item, downloaded: item.size, total: item.size)
        }
        currentIndex += 1
        downloadNext()
    }

    func didReceiveDownloadProgress(for file: ICCameraFile, downloadedBytes: off_t, maxBytes: off_t) {
        guard let item = currentItem else { return }
        let downloaded = Int64(downloadedBytes)
        if downloaded - currentDownloaded >= 16 * 1024 * 1024 || downloaded == Int64(maxBytes) {
            currentDownloaded = downloaded
            logProgress("progress", item: item, downloaded: downloaded, total: Int64(maxBytes))
        }
    }

    func logProgress(_ event: String, item: SelectionItem, downloaded: Int64, total: Int64) {
        let payload: [String: Any] = [
            "event": event,
            "index": currentIndex + 1,
            "total_files": selection.items.count,
            "path": item.path,
            "downloaded": downloaded,
            "total": total
        ]
        if let data = try? JSONSerialization.data(withJSONObject: payload),
           let text = String(data: data, encoding: .utf8) {
            FileHandle.standardError.write((text + "\n").data(using: .utf8)!)
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

if CommandLine.arguments.count < 3 {
    print("[]")
    exit(1)
}
let selectionURL = URL(fileURLWithPath: CommandLine.arguments[1])
let destRoot = URL(fileURLWithPath: CommandLine.arguments[2])
let data = try Data(contentsOf: selectionURL)
let selection = try JSONDecoder().decode(Selection.self, from: data)
Downloader(selection: selection, destRoot: destRoot).run()
`
