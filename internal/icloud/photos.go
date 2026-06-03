package icloud

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ckBaseURL returns the CloudKit database endpoint for the session.
// Partition 231 → p231-ckdatabasews.icloud.com(.cn)
func (c *Client) ckBaseURL() string {
	tld := "com"
	if c.domain == "cn" {
		tld = "com.cn"
	}
	part := c.Session.Partition
	if part == 0 {
		part = 1
	}
	return fmt.Sprintf("https://p%d-ckdatabasews.icloud.%s", part, tld)
}

// ckURL builds a full CloudKit endpoint URL.
func (c *Client) ckURL(path string) string {
	return c.ckBaseURL() +
		"/database/1/com.apple.photos.cloud/production/private" +
		path +
		"?ckjsBuildVersion=2306ProjectDev" +
		"&ckjsVersion=2.6.2" +
		"&clientBuildNumber=2306Hotfix6" +
		"&clientMasteringNumber=2306Hotfix6" +
		"&dsid=" + c.Session.DSID
}

// ckHeaders returns headers for CloudKit API calls.
func (c *Client) ckHeaders() map[string]string {
	origin := "https://www.icloud.com"
	if c.domain == "cn" {
		origin = "https://www.icloud.com.cn"
	}
	h := map[string]string{
		"Origin":  origin,
		"Referer": origin + "/",
	}
	// Inject session cookies.
	setupURL, _ := url.Parse(c.ep.setup)
	for _, ck := range c.jar.Cookies(setupURL) {
		h["Cookie"] = appendCookie(h["Cookie"], ck)
	}
	// Also inject from saved session cookies (after loading from disk).
	if len(c.Session.Cookies) > 0 && h["Cookie"] == "" {
		var parts []string
		for _, sc := range c.Session.Cookies {
			parts = append(parts, sc.Name+"="+sc.Value)
		}
		h["Cookie"] = strings.Join(parts, "; ")
	}
	return h
}

func appendCookie(existing string, ck *http.Cookie) string {
	part := ck.Name + "=" + ck.Value
	if existing == "" {
		return part
	}
	return existing + "; " + part
}

// AssetRecord represents a photo/video asset from CloudKit.
type AssetRecord struct {
	RecordName string
	Filename   string
	FileSize   int64
	CreatedAt  time.Time
	// download info
	DownloadURL   string
	DownloadToken string
}

// QueryByFilenames searches the iCloud Photos library for assets matching
// any of the given filenames.  Returns matched AssetRecords.
func (c *Client) QueryByFilenames(filenames []string) ([]AssetRecord, error) {
	// Build filter: filenameEnc EQUALS base64(filename) for each target.
	// CloudKit supports OR via multiple filters with the same field.
	// We query each name individually and merge results.
	var results []AssetRecord
	seen := map[string]bool{}
	for _, name := range filenames {
		records, err := c.queryByFilename(name)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			if !seen[r.RecordName] {
				seen[r.RecordName] = true
				results = append(results, r)
			}
		}
	}
	return results, nil
}

type ckFilter struct {
	Comparator string      `json:"comparator"`
	FieldName  string      `json:"fieldName"`
	FieldValue interface{} `json:"fieldValue"`
}

type ckQuery struct {
	RecordType string     `json:"recordType"`
	FilterBy   []ckFilter `json:"filterBy"`
}

type ckQueryRequest struct {
	Query        ckQuery  `json:"query"`
	ZoneID       ckZoneID `json:"zoneID"`
	ResultsLimit int      `json:"resultsLimit"`
	DesiredKeys  []string `json:"desiredKeys"`
}

type ckZoneID struct {
	ZoneName  string `json:"zoneName"`
	OwnerName string `json:"ownerName"`
}

type ckQueryResponse struct {
	Records []ckRecord `json:"records"`
}

type ckRecord struct {
	RecordName string                     `json:"recordName"`
	RecordType string                     `json:"recordType"`
	Fields     map[string]json.RawMessage `json:"fields"`
}

func (c *Client) queryByFilename(filename string) ([]AssetRecord, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(filename))
	req := ckQueryRequest{
		Query: ckQuery{
			RecordType: "CPLAsset",
			FilterBy: []ckFilter{
				{
					Comparator: "EQUALS",
					FieldName:  "filenameEnc",
					FieldValue: map[string]interface{}{
						"value": encoded,
						"type":  "BYTES",
					},
				},
			},
		},
		ZoneID: ckZoneID{
			ZoneName:  "PrimarySync",
			OwnerName: "_defaultOwner",
		},
		ResultsLimit: 20,
		DesiredKeys: []string{
			"filenameEnc", "resOriginalRes", "resJPEGFullRes",
			"resOriginalWidth", "resOriginalHeight",
			"assetDate", "addedDate", "filesize",
		},
	}

	path := "/records/query"
	headers := c.ckHeaders()
	headers["Content-Type"] = "application/json"

	resp, err := c.do("POST", c.ckURL(path), req, headers)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("CloudKit query returned %d: %s", resp.StatusCode, body)
	}

	var ckResp ckQueryResponse
	if err := json.Unmarshal(body, &ckResp); err != nil {
		return nil, fmt.Errorf("parsing CloudKit response: %w", err)
	}

	var results []AssetRecord
	for _, rec := range ckResp.Records {
		ar := AssetRecord{RecordName: rec.RecordName}

		// Decode filenameEnc.
		if raw, ok := rec.Fields["filenameEnc"]; ok {
			var f struct{ Value string }
			if json.Unmarshal(raw, &f) == nil {
				b, _ := base64.StdEncoding.DecodeString(f.Value)
				ar.Filename = string(b)
			}
		}
		// File size.
		if raw, ok := rec.Fields["filesize"]; ok {
			var f struct{ Value int64 }
			_ = json.Unmarshal(raw, &f)
			ar.FileSize = f.Value
		}
		// Date.
		if raw, ok := rec.Fields["assetDate"]; ok {
			var f struct{ Value int64 }
			if json.Unmarshal(raw, &f) == nil {
				ar.CreatedAt = time.UnixMilli(f.Value)
			}
		}
		// Download resource (prefer original, fall back to JPEG full).
		for _, key := range []string{"resOriginalRes", "resJPEGFullRes"} {
			if raw, ok := rec.Fields[key]; ok {
				var res struct {
					Value struct {
						DownloadURL string `json:"downloadURL"`
					}
				}
				if json.Unmarshal(raw, &res) == nil && res.Value.DownloadURL != "" {
					ar.DownloadURL = res.Value.DownloadURL
					break
				}
			}
		}
		results = append(results, ar)
	}
	return results, nil
}

// DownloadAsset downloads an AssetRecord to destDir/filename.
// Returns the path of the saved file.
func (c *Client) DownloadAsset(ar AssetRecord, destDir string) (string, error) {
	if ar.DownloadURL == "" {
		return "", fmt.Errorf("no download URL for %s", ar.Filename)
	}

	// The download URL may need auth cookies.
	req, err := http.NewRequest("GET", ar.DownloadURL, nil)
	if err != nil {
		return "", err
	}
	// Apply session cookies.
	setupURL, _ := url.Parse(c.ep.setup)
	for _, ck := range c.jar.Cookies(setupURL) {
		req.AddCookie(ck)
	}
	if len(c.Session.Cookies) > 0 {
		for _, sc := range c.Session.Cookies {
			req.AddCookie(&http.Cookie{Name: sc.Name, Value: sc.Value}) //nolint:gosec // G124: iCloud third-party cookie
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned %d for %s", resp.StatusCode, ar.Filename)
	}

	destPath := filepath.Join(destDir, ar.Filename)
	f, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return destPath, nil
}
