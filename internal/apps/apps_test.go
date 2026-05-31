package apps

import "testing"

func TestParsePlistApps(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><array><dict>
<key>StaticDiskUsage</key><integer>100</integer>
<key>CFBundleDisplayName</key><string>WeChat</string>
<key>CFBundleIdentifier</key><string>com.tencent.xin</string>
<key>DynamicDiskUsage</key><integer>900</integer>
</dict></array></plist>`)
	apps, err := parsePlistApps(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("len = %d, want 1", len(apps))
	}
	if apps[0].TotalSize != 1000 || apps[0].Name != "WeChat" {
		t.Fatalf("app = %+v", apps[0])
	}
}
