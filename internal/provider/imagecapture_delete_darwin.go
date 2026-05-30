//go:build darwin

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/chenhg5/imole/internal/syscmd"
)

// DeleteRequest specifies a single file path (relative, as returned by scan) to delete from the device.
type DeleteRequest struct {
	Path string `json:"path"`
}

// DeleteResult reports the outcome for a single deletion attempt.
type DeleteResult struct {
	Path    string `json:"path"`
	Deleted bool   `json:"deleted"`
	Error   string `json:"error,omitempty"`
}

type imageCaptureDeleteSelection struct {
	Items []DeleteRequest `json:"items"`
}

// DeleteImageCapture deletes the given paths from the connected iPhone using ImageCaptureCore.
// Only paths that appear in the device's DCIM catalog can be deleted.
// On newer iOS versions the device may display a confirmation prompt to the user.
func DeleteImageCapture(ctx context.Context, requests []DeleteRequest) ([]DeleteResult, error) {
	if len(requests) == 0 {
		return nil, nil
	}
	swift, err := syscmd.LookPath("swift")
	if err != nil {
		return nil, fmt.Errorf("swift not found for ImageCaptureCore delete helper")
	}

	helper, err := os.CreateTemp("", "imole-imagecapture-delete-*.swift")
	if err != nil {
		return nil, err
	}
	defer os.Remove(helper.Name())
	if _, err := helper.WriteString(imageCaptureDeleteSwiftSource); err != nil {
		_ = helper.Close()
		return nil, err
	}
	if err := helper.Close(); err != nil {
		return nil, err
	}

	selectionFile, err := os.CreateTemp("", "imole-imagecapture-delete-sel-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(selectionFile.Name())

	selection := imageCaptureDeleteSelection{Items: requests}
	if err := json.NewEncoder(selectionFile).Encode(selection); err != nil {
		_ = selectionFile.Close()
		return nil, err
	}
	if err := selectionFile.Close(); err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(runCtx, swift, helper.Name(), selectionFile.Name())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ImageCaptureCore delete helper failed: %s", stderr.String())
	}

	var results []DeleteResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("ImageCaptureCore delete helper returned invalid JSON: %w: %s", err, stdout.String())
	}
	return results, nil
}

const imageCaptureDeleteSwiftSource = `
import Foundation
import ImageCaptureCore

struct Selection: Decodable {
    let items: [SelectionItem]
}

struct SelectionItem: Decodable {
    let path: String
}

struct ResultItem: Encodable {
    let path: String
    let deleted: Bool
    let error: String?
}

final class Deleter: NSObject, ICDeviceBrowserDelegate, ICDeviceDelegate, ICCameraDeviceDelegate, ICCameraDeviceDeletionDelegate {
    let browser = ICDeviceBrowser()
    let selection: Selection
    var catalogReady = false
    var done = false
    var catalog: [String: ICCameraFile] = [:]
    // Maps ObjectIdentifier(ICCameraFile) -> original path string for per-file tracking.
    var pendingByID: [ObjectIdentifier: String] = [:]
    // Tracks paths removed from the catalog via didRemove callback (= confirmed deleted).
    var removedPaths: Set<String> = []
    var results: [ResultItem] = []
    var camera: ICCameraDevice?
    var deleteIssued = false

    init(selection: Selection) {
        self.selection = selection
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
        guard catalogReady else {
            for item in selection.items {
                results.append(ResultItem(path: item.path, deleted: false, error: "timeout: no device catalog within 60s"))
            }
            browser.stop()
            emit()
            return
        }

        deleteFiles()

        // Wait until the deletion callback fires or we time out.
        let deleteUntil = Date().addingTimeInterval(300)
        while !done && Date() < deleteUntil {
            RunLoop.current.run(mode: .default, before: Date().addingTimeInterval(0.2))
        }
        if !done {
            // Fill any results that were never reported.
            let reportedPaths = Set(results.map { $0.path })
            for item in selection.items where !reportedPaths.contains(item.path) {
                results.append(ResultItem(path: item.path, deleted: false, error: "delete timed out"))
            }
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

    // MARK: ICDeviceBrowserDelegate

    func deviceBrowser(_ browser: ICDeviceBrowser, didAdd device: ICDevice, moreComing: Bool) {
        device.delegate = self
        guard let cam = device as? ICCameraDevice else { return }
        camera = cam
        cam.delegate = self
        cam.requestOpenSession()
    }

    func deviceBrowser(_ browser: ICDeviceBrowser, didRemove device: ICDevice, moreGoing: Bool) {}
    func didRemove(_ device: ICDevice) {}
    func device(_ device: ICDevice, didOpenSessionWithError error: Error?) {}
    func device(_ device: ICDevice, didCloseSessionWithError error: Error?) {}

    // MARK: ICCameraDeviceDelegate

    func deviceDidBecomeReady(withCompleteContentCatalog device: ICCameraDevice) {
        catalog.removeAll()
        collect(items: device.contents ?? [], prefix: "")
        catalogReady = true
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

    // Called when files are removed from the device catalog (confirmed deleted on device).
    func cameraDevice(_ camera: ICCameraDevice, didRemove items: [ICCameraItem]) {
        for item in items {
            guard let file = item as? ICCameraFile else { continue }
            if let path = pendingByID[ObjectIdentifier(file)] {
                removedPaths.insert(path)
            }
        }
    }

    // MARK: Delete

    func deleteFiles() {
        var filesToDelete: [ICCameraFile] = []

        for item in selection.items {
            if let file = catalog[item.path] {
                filesToDelete.append(file)
                pendingByID[ObjectIdentifier(file)] = item.path
            } else {
                results.append(ResultItem(path: item.path, deleted: false, error: "not found in device catalog"))
            }
        }

        if filesToDelete.isEmpty {
            done = true
            return
        }

        deleteIssued = true
        camera?.requestDeleteFiles(
            filesToDelete,
            deletionDelegate: self,
            didDeleteSelector: #selector(didDeleteSomeOrAllFiles(_:backData:error:contextInfo:)),
            contextInfo: nil
        )
    }

    // MARK: ICCameraDeviceDeletionDelegate

    // Called once when the batch delete operation completes (success or failure).
    @objc func didDeleteSomeOrAllFiles(
        _ fileURLs: [URL]?,
        backData: [AnyHashable: Any]?,
        error: Error?,
        contextInfo: UnsafeMutableRawPointer?
    ) {
        // After the callback, reconcile pending items against the removedPaths set.
        let reportedPaths = Set(results.map { $0.path })
        for item in selection.items {
            if reportedPaths.contains(item.path) { continue }
            let wasRemoved = removedPaths.contains(item.path)
            if wasRemoved {
                results.append(ResultItem(path: item.path, deleted: true, error: nil))
            } else if let err = error {
                // If there was an error and the file wasn't confirmed removed, report the error.
                results.append(ResultItem(path: item.path, deleted: false, error: err.localizedDescription))
            } else {
                // No per-file remove event and no error — treat as deleted (some iOS versions
                // don't fire didRemove for each file individually).
                results.append(ResultItem(path: item.path, deleted: true, error: nil))
            }
        }
        done = true
    }

    func cameraDeviceDidChangeCapability(_ camera: ICCameraDevice) {}
    func cameraDevice(_ camera: ICCameraDevice, didAdd items: [ICCameraItem]) {}
    func cameraDevice(_ camera: ICCameraDevice, didReceivePTPEvent eventData: Data) {}
    func cameraDevice(_ camera: ICCameraDevice, didRenameItems items: [ICCameraItem]) {}
    func cameraDevice(_ camera: ICCameraDevice, didReceiveThumbnail thumbnail: CGImage?, for item: ICCameraItem, error: Error?) {}
    func cameraDevice(_ camera: ICCameraDevice, didReceiveMetadata metadata: [AnyHashable: Any]?, for item: ICCameraItem, error: Error?) {}
    func cameraDeviceDidRemoveAccessRestriction(_ device: ICDevice) {}
    func cameraDeviceDidEnableAccessRestriction(_ device: ICDevice) {}
}

if CommandLine.arguments.count < 2 {
    print("[]")
    exit(1)
}
let selectionURL = URL(fileURLWithPath: CommandLine.arguments[1])
let data = try Data(contentsOf: selectionURL)
let selection = try JSONDecoder().decode(Selection.self, from: data)
Deleter(selection: selection).run()
`
