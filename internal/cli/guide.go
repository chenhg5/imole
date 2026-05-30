package cli

import (
	"context"
	"fmt"
)

func (a *App) runGuide(_ context.Context, args []string) error {
	topic := "all"
	if len(args) > 0 {
		topic = args[0]
	}
	switch topic {
	case "photos", "photo":
		fmt.Fprint(a.out, photosGuide)
	case "wechat":
		fmt.Fprint(a.out, wechatGuide)
	case "system", "system-data":
		fmt.Fprint(a.out, systemGuide)
	default:
		fmt.Fprint(a.out, fullGuide)
	}
	return nil
}

const fullGuide = `iPhone slimming guide

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
  imole videos --top 50
  imole backup --source /path/to/DCIM --to /path/to/backup --only videos --older-than 90d

After verification, delete imported items with Apple Image Capture or Photos,
then empty Recently Deleted on the iPhone.
`

const wechatGuide = `WeChat cleanup

Open WeChat > Me > Settings > General > Storage.
Clear cache before deleting chat history. Review large chats manually.
iMole cannot inspect or delete WeChat private app storage over USB.
`

const systemGuide = `System Data cleanup

iOS does not expose a supported USB API for third-party tools to clean System Data.
Practical steps: restart iPhone, update iOS, remove failed update packages if shown,
and clear large app caches from each app's own settings.
`
