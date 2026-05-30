package media

func TopVideos(items []Item, n int) []Item {
	if n <= 0 || n > len(items) {
		n = len(items)
	}
	out := make([]Item, 0, n)
	for _, item := range items {
		if item.IsVideo() {
			out = append(out, item)
			if len(out) == n {
				break
			}
		}
	}
	return out
}
