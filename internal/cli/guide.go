package cli

import (
	"context"
	"fmt"
)

func (a *App) runGuide(_ context.Context, args []string) int {
	topic := ""
	if len(args) > 0 {
		topic = args[0]
	}
	switch topic {
	case "analysis", "analyze", "agent", "playbook":
		fmt.Fprint(a.out, analysisGuide)
	case "photos", "photo":
		fmt.Fprint(a.out, photosGuide)
	case "wechat":
		fmt.Fprint(a.out, wechatGuide)
	case "system", "system-data":
		fmt.Fprint(a.out, systemGuide)
	case "trust", "pair", "pairing":
		fmt.Fprint(a.out, trustGuide)
	default:
		fmt.Fprint(a.out, fullGuide)
	}
	return ExitSuccess
}

const fullGuide = `iPhone slimming guide

For agents and scripts:
   Run: imole guide analysis
   Then follow the pressure -> source -> action workflow before proposing cleanup.

1. Photos and videos
   Run: imole scan
   Run: imole backup --to /path/to/backup --only videos --older-than 90d
   Delete only after backup verification, then empty Photos > Recently Deleted.

2. WeChat
   WeChat > Me > Settings > General > Storage.
   Clear cache first, then review large chats.

3. App downloads
   Settings > General > iPhone Storage.
   Review video, music, podcast, map, and cloud-drive offline downloads.

4. System Data
   Restart iPhone, keep iOS updated, remove failed update packages if visible.
   iMole cannot directly clean iOS System Data over USB.
`

const photosGuide = `Photos and videos

Use iMole to find and back up large media first:
  imole scan
  imole scan --top 50 --only videos
  imole backup --source /path/to/DCIM --to /path/to/backup --only videos --older-than 90d

After verification, delete imported items with Apple Image Capture or Photos,
then empty Recently Deleted on the iPhone.
`

const analysisGuide = `iMole storage analysis playbook

Use this when a human asks: "what can I optimize on my iPhone?"

1. Measure storage pressure
   Run:
     imole doctor --json --fields device.name,device.product_type,device.storage.total_data_capacity,device.storage.amount_data_available,device.storage.free_percent

   If device.storage is missing or trusted=false:
     idevicepair pair
     idevicepair validate

   Keep the iPhone unlocked and tap Trust on the device. ImageCapture can sometimes read photos even when libimobiledevice pairing is not trusted yet.

   Classify free space:
     <5%     critical  -> target immediate reclaim
     5-10%   high      -> plan enough cleanup to reach at least 15% free
     10-20%  moderate  -> use conservative filters first
     >20%    low       -> diagnose only unless the user asks to delete

2. Find the largest storage buckets
   Run:
     imole scan --summary --json --fields device.storage.free_percent,media.total_size,media.photo_size,media.video_size,apps.total_size,top_video

   Interpret:
     - If videos are large enough to meet the target, prefer videos first.
     - If app storage dominates, run app ranking and give in-app cleanup guidance.
     - App sizes are estimates from iOS installation_proxy and can underreport apps using shared containers.

3. Inspect candidates before proposing action
   Run:
     imole scan --top 20 --only videos --json
     imole scan apps --top 20 --json

   Recommend the smallest-risk media filter that reaches the target:
     old videos > large videos > all videos > photos

   Typical filters:
     --only videos --older-than 1y
     --only videos --older-than 6m
     --only videos --large-than 500MB

4. Present a concrete plan
   Good answer shape:
     - Current pressure: free percent and free size.
     - Main contributors: media, videos, apps, top video/app.
     - Recommended first action: exact backup command and expected reclaim.
     - Manual app cleanup: only for apps iMole cannot safely clean directly.
     - Risk notes: iCloud Photos and Recently Deleted.

5. Preview side effects only on side-effecting commands
   Do not add --dry-run to scan, scan apps, doctor, report, history, schema, or guide.
   Use:
     imole backup --to ~/imole-backup [filters] --dry-run
     imole clean --manifest ~/imole-backup/manifest.json --dry-run

6. Safety rule
   Never recommend deletion until backup completed and report shows verified files.
   Run:
     imole report --manifest ~/imole-backup/manifest.json --json
`

const wechatGuide = `WeChat cleanup

Open WeChat > Me > Settings > General > Storage.
Clear cache before deleting chat history. Review large chats manually.
iMole cannot inspect or delete WeChat private app storage over USB.
`

const trustGuide = `iPhone USB trust / pairing

iMole uses two different Apple-facing paths:
  - ImageCaptureCore can read camera roll media on macOS.
  - libimobiledevice reads device metadata such as total capacity and free space.

It is possible for media scan to work while device storage is unavailable.
That usually means ImageCapture can see photos, but libimobiledevice is not paired.

To trigger the iPhone "Trust This Computer" prompt:
  1. Connect the iPhone by USB.
  2. Unlock the iPhone and keep it on the home screen.
  3. Run:
       idevicepair pair
  4. Tap Trust on the iPhone and enter the passcode if asked.
  5. Verify:
       idevicepair validate
       imole doctor

If no prompt appears:
  - Unplug and reconnect the cable.
  - Try a data-capable USB cable and a direct Mac port.
  - On iPhone: Settings > General > Transfer or Reset iPhone > Reset > Reset Location & Privacy, then run idevicepair pair again.
`

const systemGuide = `System Data cleanup

iOS does not expose a supported USB API for third-party tools to clean System Data.
Practical steps: restart iPhone, update iOS, remove failed update packages if shown,
and clear large app caches from each app's own settings.
`
