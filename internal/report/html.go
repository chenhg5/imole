package report

import (
	"fmt"
	"html/template"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/backup"
	"github.com/chenhg5/imole/internal/human"
)

// HTMLData holds all data needed to render the HTML report.
type HTMLData struct {
	GeneratedAt   string
	Device        string
	ManifestDate  string
	TotalFreed    string
	TotalFreedRaw int64
	TotalFiles    int
	PhotoFiles    int
	VideoFiles    int
	OtherFiles    int
	PhotoSize     int64
	VideoSize     int64
	OtherSize     int64
	PhotoSizeStr  string
	VideoSizeStr  string
	OtherSizeStr  string
	PhotoPct      float64
	VideoPct      float64
	OtherPct      float64
	TopFiles      []TopFile
	MonthGroups   []MonthGroup
	VerifyHealth  string // "99.8%" or "" if not verified
	ShareText     string
}

type TopFile struct {
	Name    string
	Size    string
	SizeRaw int64
	Kind    string
	Date    string
	BarPct  float64
}

type MonthGroup struct {
	Label   string
	Files   int
	Size    string
	SizeRaw int64
	BarPct  float64
}

// BuildHTMLData constructs HTMLData from a manifest and optional verify result.
func BuildHTMLData(manifest backup.Manifest, verify *VerifyResult) HTMLData {
	d := HTMLData{
		GeneratedAt:  time.Now().Format("2006-01-02 15:04"),
		Device:       manifest.Device,
		ManifestDate: manifest.CreatedAt.Format("2006-01-02"),
	}

	// Aggregate by kind and month
	monthMap := make(map[string]*MonthGroup)
	type fileEntry struct {
		name    string
		size    int64
		kind    string
		modTime time.Time
	}
	var allFiles []fileEntry

	for _, f := range manifest.Files {
		if !f.Verified {
			continue
		}
		d.TotalFiles++
		d.TotalFreedRaw += f.Size
		switch f.Kind {
		case "photo":
			d.PhotoFiles++
			d.PhotoSize += f.Size
		case "video":
			d.VideoFiles++
			d.VideoSize += f.Size
		default:
			d.OtherFiles++
			d.OtherSize += f.Size
		}
		// Month grouping
		key := f.ModTime.Format("2006-01")
		if _, ok := monthMap[key]; !ok {
			monthMap[key] = &MonthGroup{Label: f.ModTime.Format("2006-01")}
		}
		monthMap[key].Files++
		monthMap[key].SizeRaw += f.Size

		allFiles = append(allFiles, fileEntry{
			name:    destBaseName(f.DestRel),
			size:    f.Size,
			kind:    f.Kind,
			modTime: f.ModTime,
		})
	}

	d.TotalFreed = human.Bytes(d.TotalFreedRaw)
	d.PhotoSizeStr = human.Bytes(d.PhotoSize)
	d.VideoSizeStr = human.Bytes(d.VideoSize)
	d.OtherSizeStr = human.Bytes(d.OtherSize)
	if d.TotalFreedRaw > 0 {
		d.PhotoPct = math.Round(float64(d.PhotoSize) * 100 / float64(d.TotalFreedRaw))
		d.VideoPct = math.Round(float64(d.VideoSize) * 100 / float64(d.TotalFreedRaw))
		d.OtherPct = math.Round(float64(d.OtherSize) * 100 / float64(d.TotalFreedRaw))
	}

	// Top 10 files by size
	sort.Slice(allFiles, func(i, j int) bool { return allFiles[i].size > allFiles[j].size })
	maxSize := int64(1)
	if len(allFiles) > 0 {
		maxSize = allFiles[0].size
	}
	limit := 10
	if len(allFiles) < limit {
		limit = len(allFiles)
	}
	for _, f := range allFiles[:limit] {
		d.TopFiles = append(d.TopFiles, TopFile{
			Name:    f.name,
			Size:    human.Bytes(f.size),
			SizeRaw: f.size,
			Kind:    f.kind,
			Date:    f.modTime.Format("2006-01-02"),
			BarPct:  float64(f.size) * 100 / float64(maxSize),
		})
	}

	// Month groups sorted by date
	months := make([]MonthGroup, 0, len(monthMap))
	for _, mg := range monthMap {
		mg.Size = human.Bytes(mg.SizeRaw)
		months = append(months, *mg)
	}
	sort.Slice(months, func(i, j int) bool { return months[i].Label > months[j].Label })
	maxMonthSize := int64(1)
	for _, m := range months {
		if m.SizeRaw > maxMonthSize {
			maxMonthSize = m.SizeRaw
		}
	}
	for i := range months {
		months[i].BarPct = float64(months[i].SizeRaw) * 100 / float64(maxMonthSize)
	}
	if len(months) > 18 {
		months = months[:18]
	}
	d.MonthGroups = months

	// Verify health
	if verify != nil {
		d.VerifyHealth = fmt.Sprintf("%.1f%%", verify.HealthPct)
	}

	// Share text
	deviceStr := "my iPhone"
	if d.Device != "" {
		deviceStr = d.Device
	}
	d.ShareText = fmt.Sprintf("I freed %s from %s using iMole 🕳️", d.TotalFreed, deviceStr)

	return d
}

func destBaseName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

// GenerateHTML renders the full HTML report as a string.
func GenerateHTML(data HTMLData) (string, error) {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"mul":     func(a, b float64) float64 { return a * b },
		"add":     func(a, b float64) float64 { return a + b },
		"dashlen": func(pct float64) string { return fmt.Sprintf("%.2f", pct*339.3/100) },
	}).Parse(htmlTemplate)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>iMole Report — {{.TotalFreed}} freed</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#07080f;--card:#0f1018;--border:#1e2030;
  --cyan:#00e5ff;--purple:#b966f5;--pink:#ff4d8f;--green:#00e676;--yellow:#ffca28;
  --text:#e8eaf0;--muted:#6b7280;--radius:14px;
}
body{background:var(--bg);color:var(--text);font-family:-apple-system,BlinkMacSystemFont,'SF Pro Display','Segoe UI',sans-serif;min-height:100vh;padding:0 0 60px}
a{color:var(--cyan);text-decoration:none}

/* ── HERO ── */
.hero{
  text-align:center;padding:72px 24px 48px;
  background:radial-gradient(ellipse 80% 60% at 50% -10%, #1a0a3a 0%, transparent 70%);
  position:relative;overflow:hidden;
}
.hero::before{
  content:'';position:absolute;inset:0;
  background:radial-gradient(circle at 80% 50%, rgba(0,229,255,.06) 0%, transparent 60%),
             radial-gradient(circle at 20% 50%, rgba(185,102,245,.06) 0%, transparent 60%);
  pointer-events:none;
}
.hero-eyebrow{font-size:13px;letter-spacing:.14em;text-transform:uppercase;color:var(--muted);margin-bottom:16px}
.hero-number{
  font-size:clamp(72px,14vw,128px);font-weight:800;line-height:1;letter-spacing:-.03em;
  background:linear-gradient(135deg, var(--cyan) 0%, var(--purple) 50%, var(--pink) 100%);
  -webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;
  filter:drop-shadow(0 0 40px rgba(0,229,255,.25));
  margin-bottom:8px;
}
.hero-sub{font-size:22px;color:var(--muted);font-weight:400}
.hero-meta{margin-top:20px;font-size:13px;color:var(--muted)}
.hero-meta span{color:var(--text);font-weight:500}
.badge{
  display:inline-block;margin:4px 6px;padding:4px 12px;border-radius:20px;font-size:12px;
  background:rgba(255,255,255,.07);border:1px solid var(--border);
}

/* ── LAYOUT ── */
.page{max-width:960px;margin:0 auto;padding:0 20px}
.section{margin-top:40px}
.section-title{font-size:11px;letter-spacing:.12em;text-transform:uppercase;color:var(--muted);margin-bottom:16px;padding-bottom:8px;border-bottom:1px solid var(--border)}

/* ── CARDS ── */
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px}
.card{background:var(--card);border:1px solid var(--border);border-radius:var(--radius);padding:20px}
.card-label{font-size:11px;color:var(--muted);text-transform:uppercase;letter-spacing:.1em;margin-bottom:8px}
.card-value{font-size:28px;font-weight:700;letter-spacing:-.02em}
.card-sub{font-size:12px;color:var(--muted);margin-top:4px}
.c-cyan{color:var(--cyan)}
.c-purple{color:var(--purple)}
.c-pink{color:var(--pink)}
.c-green{color:var(--green)}
.c-yellow{color:var(--yellow)}

/* ── DONUT ── */
.breakdown{display:grid;grid-template-columns:auto 1fr;gap:32px;align-items:center;background:var(--card);border:1px solid var(--border);border-radius:var(--radius);padding:28px}
.donut-wrap{position:relative;width:140px;height:140px;flex-shrink:0}
.donut-wrap svg{transform:rotate(-90deg)}
.donut-center{position:absolute;inset:0;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center}
.donut-center-num{font-size:22px;font-weight:700}
.donut-center-lbl{font-size:10px;color:var(--muted);text-transform:uppercase;letter-spacing:.08em}
.legend{display:flex;flex-direction:column;gap:14px}
.legend-item{display:flex;align-items:center;gap:10px}
.legend-dot{width:10px;height:10px;border-radius:50%;flex-shrink:0}
.legend-label{font-size:13px;color:var(--muted);min-width:50px}
.legend-value{font-size:15px;font-weight:600;margin-left:auto}
.legend-pct{font-size:12px;color:var(--muted);width:36px;text-align:right}

/* ── TOP FILES ── */
.file-list{display:flex;flex-direction:column;gap:6px}
.file-row{background:var(--card);border:1px solid var(--border);border-radius:10px;padding:12px 16px;position:relative;overflow:hidden}
.file-row::before{content:'';position:absolute;left:0;top:0;bottom:0;border-radius:10px 0 0 10px;width:3px}
.file-row.photo::before{background:var(--cyan)}
.file-row.video::before{background:var(--purple)}
.file-row.other::before{background:var(--muted)}
.file-bar{position:absolute;left:0;top:0;bottom:0;opacity:.06;border-radius:10px}
.file-bar.photo{background:var(--cyan)}
.file-bar.video{background:var(--purple)}
.file-info{display:flex;align-items:center;gap:12px;position:relative}
.file-name{font-size:13px;font-weight:500;flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.file-size{font-size:13px;font-weight:700;white-space:nowrap}
.file-meta{font-size:11px;color:var(--muted);white-space:nowrap}
.file-kind{font-size:10px;padding:2px 7px;border-radius:4px;text-transform:uppercase;letter-spacing:.06em;font-weight:600}
.file-kind.photo{background:rgba(0,229,255,.12);color:var(--cyan)}
.file-kind.video{background:rgba(185,102,245,.12);color:var(--purple)}

/* ── MONTH CHART ── */
.month-chart{display:flex;flex-direction:column;gap:8px}
.month-row{display:grid;grid-template-columns:80px 1fr 80px 56px;gap:10px;align-items:center;font-size:13px}
.month-label{color:var(--muted);font-variant-numeric:tabular-nums}
.month-bar-wrap{background:rgba(255,255,255,.04);border-radius:4px;height:8px;overflow:hidden}
.month-bar{height:100%;border-radius:4px;background:linear-gradient(90deg,var(--cyan),var(--purple));transition:width .4s}
.month-size{text-align:right;font-weight:600;font-variant-numeric:tabular-nums}
.month-count{text-align:right;color:var(--muted);font-size:11px}

/* ── SHARE CARD ── */
.share-outer{margin-top:40px}
.share-card{
  background:linear-gradient(135deg,#0d0a1f 0%,#0a1520 50%,#0d0a1f 100%);
  border:1px solid rgba(0,229,255,.2);border-radius:20px;padding:40px 36px;
  position:relative;overflow:hidden;
}
.share-card::before{
  content:'';position:absolute;inset:0;
  background:radial-gradient(circle at 10% 90%, rgba(185,102,245,.12) 0%, transparent 50%),
             radial-gradient(circle at 90% 10%, rgba(0,229,255,.10) 0%, transparent 50%);
  pointer-events:none;
}
.share-logo{font-size:13px;font-weight:700;letter-spacing:.06em;color:var(--muted);margin-bottom:20px;position:relative}
.share-logo span{color:var(--cyan)}
.share-headline{
  font-size:clamp(24px,4vw,36px);font-weight:800;line-height:1.2;
  background:linear-gradient(135deg,#fff 0%,rgba(255,255,255,.7) 100%);
  -webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;
  position:relative;margin-bottom:16px;
}
.share-stats{display:flex;gap:24px;flex-wrap:wrap;position:relative}
.share-stat-item{display:flex;flex-direction:column}
.share-stat-val{font-size:20px;font-weight:700}
.share-stat-lbl{font-size:11px;color:var(--muted);text-transform:uppercase;letter-spacing:.08em}
.share-corner{
  position:absolute;right:36px;top:50%;transform:translateY(-50%);
  font-size:64px;opacity:.15;
}
.share-tag{margin-top:24px;font-size:12px;color:var(--muted);position:relative}
.share-tag a{color:var(--cyan)}

/* ── HEALTH ── */
.health-bar-wrap{background:rgba(255,255,255,.05);border-radius:6px;height:12px;overflow:hidden;margin-top:8px}
.health-bar{height:100%;border-radius:6px;background:linear-gradient(90deg,var(--green),var(--cyan))}
.health-issues{margin-top:12px;font-size:12px;color:var(--muted)}
.health-issues li{margin-top:4px;font-family:monospace;color:var(--pink)}

/* ── FOOTER ── */
.footer{text-align:center;margin-top:60px;font-size:12px;color:var(--muted);padding:0 20px}

@media(max-width:600px){
  .breakdown{grid-template-columns:1fr}
  .donut-wrap{margin:0 auto}
  .month-row{grid-template-columns:60px 1fr 70px}
  .month-count{display:none}
  .share-corner{display:none}
}
</style>
</head>
<body>

<!-- HERO -->
<div class="hero">
  <div class="hero-eyebrow">iMole Storage Report</div>
  <div class="hero-number">{{.TotalFreed}}</div>
  <div class="hero-sub">freed from iPhone</div>
  <div class="hero-meta" style="margin-top:16px">
    <span class="badge">📅 {{.ManifestDate}}</span>
    <span class="badge">📁 {{.TotalFiles}} files</span>
    {{if .Device}}<span class="badge">📱 {{.Device}}</span>{{end}}
    {{if .VerifyHealth}}<span class="badge">🛡 {{.VerifyHealth}} healthy</span>{{end}}
  </div>
</div>

<div class="page">

<!-- STAT CARDS -->
<div class="section">
  <div class="section-title">Summary</div>
  <div class="cards">
    <div class="card">
      <div class="card-label">Photos</div>
      <div class="card-value c-cyan">{{.PhotoSizeStr}}</div>
      <div class="card-sub">{{.PhotoFiles}} files</div>
    </div>
    <div class="card">
      <div class="card-label">Videos</div>
      <div class="card-value c-purple">{{.VideoSizeStr}}</div>
      <div class="card-sub">{{.VideoFiles}} files</div>
    </div>
    <div class="card">
      <div class="card-label">Other</div>
      <div class="card-value" style="color:var(--muted)">{{.OtherSizeStr}}</div>
      <div class="card-sub">{{.OtherFiles}} files</div>
    </div>
    <div class="card">
      <div class="card-label">Total Freed</div>
      <div class="card-value c-green">{{.TotalFreed}}</div>
      <div class="card-sub">{{.TotalFiles}} files total</div>
    </div>
  </div>
</div>

<!-- BREAKDOWN DONUT -->
<div class="section">
  <div class="section-title">Storage Breakdown</div>
  <div class="breakdown">
    <div class="donut-wrap">
      <svg width="140" height="140" viewBox="0 0 140 140">
        <circle cx="70" cy="70" r="54" fill="none" stroke="#1e2030" stroke-width="20"/>
        {{if gt .PhotoPct 0.0}}
        <circle cx="70" cy="70" r="54" fill="none" stroke="#00e5ff" stroke-width="20"
          stroke-dasharray="{{dashlen .PhotoPct}} 339.3"
          stroke-dashoffset="0"/>
        {{end}}
        {{if gt .VideoPct 0.0}}
        <circle cx="70" cy="70" r="54" fill="none" stroke="#b966f5" stroke-width="20"
          stroke-dasharray="{{dashlen .VideoPct}} 339.3"
          stroke-dashoffset="-{{dashlen .PhotoPct}}"/>
        {{end}}
        {{if gt .OtherPct 0.0}}
        <circle cx="70" cy="70" r="54" fill="none" stroke="#6b7280" stroke-width="20"
          stroke-dasharray="{{dashlen .OtherPct}} 339.3"
          stroke-dashoffset="-{{dashlen (add .PhotoPct .VideoPct)}}"/>
        {{end}}
      </svg>
      <div class="donut-center">
        <div class="donut-center-num">{{.TotalFiles}}</div>
        <div class="donut-center-lbl">files</div>
      </div>
    </div>
    <div class="legend">
      <div class="legend-item">
        <div class="legend-dot" style="background:var(--cyan)"></div>
        <span class="legend-label">Photos</span>
        <span class="legend-value c-cyan">{{.PhotoSizeStr}}</span>
        <span class="legend-pct">{{printf "%.0f" .PhotoPct}}%</span>
      </div>
      <div class="legend-item">
        <div class="legend-dot" style="background:var(--purple)"></div>
        <span class="legend-label">Videos</span>
        <span class="legend-value c-purple">{{.VideoSizeStr}}</span>
        <span class="legend-pct">{{printf "%.0f" .VideoPct}}%</span>
      </div>
      <div class="legend-item">
        <div class="legend-dot" style="background:var(--muted)"></div>
        <span class="legend-label">Other</span>
        <span class="legend-value" style="color:var(--muted)">{{.OtherSizeStr}}</span>
        <span class="legend-pct">{{printf "%.0f" .OtherPct}}%</span>
      </div>
    </div>
  </div>
</div>

<!-- TOP FILES -->
{{if .TopFiles}}
<div class="section">
  <div class="section-title">Largest Files</div>
  <div class="file-list">
    {{range .TopFiles}}
    <div class="file-row {{.Kind}}">
      <div class="file-bar {{.Kind}}" style="width:{{printf "%.1f" .BarPct}}%"></div>
      <div class="file-info">
        <span class="file-kind {{.Kind}}">{{.Kind}}</span>
        <span class="file-name">{{.Name}}</span>
        <span class="file-meta">{{.Date}}</span>
        <span class="file-size">{{.Size}}</span>
      </div>
    </div>
    {{end}}
  </div>
</div>
{{end}}

<!-- MONTH TIMELINE -->
{{if .MonthGroups}}
<div class="section">
  <div class="section-title">By Month</div>
  <div class="month-chart">
    {{range .MonthGroups}}
    <div class="month-row">
      <span class="month-label">{{.Label}}</span>
      <div class="month-bar-wrap">
        <div class="month-bar" style="width:{{printf "%.1f" .BarPct}}%"></div>
      </div>
      <span class="month-size">{{.Size}}</span>
      <span class="month-count">{{.Files}} files</span>
    </div>
    {{end}}
  </div>
</div>
{{end}}

<!-- SHARE CARD -->
<div class="share-outer">
  <div class="section-title">Share</div>
  <div class="share-card">
    <div class="share-logo"><span>iMole</span> · iPhone Storage Cleaner</div>
    <div class="share-headline">{{.ShareText}}</div>
    <div class="share-stats">
      <div class="share-stat-item">
        <span class="share-stat-val c-cyan">{{.TotalFreed}}</span>
        <span class="share-stat-lbl">freed</span>
      </div>
      <div class="share-stat-item">
        <span class="share-stat-val c-purple">{{.TotalFiles}}</span>
        <span class="share-stat-lbl">files</span>
      </div>
      <div class="share-stat-item">
        <span class="share-stat-val c-green">{{.PhotoFiles}}</span>
        <span class="share-stat-lbl">photos</span>
      </div>
      <div class="share-stat-item">
        <span class="share-stat-val" style="color:var(--yellow)">{{.VideoFiles}}</span>
        <span class="share-stat-lbl">videos</span>
      </div>
    </div>
    <div class="share-corner">🕳️</div>
    <div class="share-tag">Generated by <a href="https://github.com/chenhg5/imole">iMole</a> · open-source iPhone slimmer</div>
  </div>
</div>

</div><!-- /page -->

<div class="footer">
  Generated {{.GeneratedAt}} · <a href="https://github.com/chenhg5/imole">github.com/chenhg5/imole</a>
</div>

</body>
</html>`
