package media

import "sort"

// TopItems returns up to n items sorted by size descending.
// kind filters: "videos", "photos", or "all" (empty string = all).
// Items passed in are assumed to already be sorted by size descending
// (as returned by the imagecapture provider), but we re-sort to be safe.
func TopItems(items []Item, kind string, n int) []Item {
	sorted := make([]Item, 0, len(items))
	for _, item := range items {
		switch kind {
		case "videos", "video":
			if !item.IsVideo() {
				continue
			}
		case "photos", "photo":
			if !item.IsPhoto() {
				continue
			}
		}
		sorted = append(sorted, item)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Size > sorted[j].Size
	})
	if n > 0 && n < len(sorted) {
		return sorted[:n]
	}
	return sorted
}

// TopVideos is kept for internal backward compatibility.
func TopVideos(items []Item, n int) []Item {
	return TopItems(items, "videos", n)
}
