package provider

import (
	"context"
	"os"

	"github.com/chenhg5/imole/internal/geo"
	"github.com/chenhg5/imole/internal/media"
	"github.com/rwcarlsen/goexif/exif"
)

func ScanFilesystem(ctx context.Context, source string, opts media.Options) (media.Result, error) {
	result, err := media.Scan(ctx, source, opts)
	if err != nil || !opts.WithMeta {
		return result, err
	}

	// Enrich items with EXIF metadata read from local files.
	for i := range result.Items {
		if ctx.Err() != nil {
			break
		}
		item := &result.Items[i]
		if item.Kind != "photo" {
			continue
		}
		readExifIntoItem(item)
	}
	return result, nil
}

func readExifIntoItem(item *media.Item) {
	f, err := os.Open(item.SourcePath)
	if err != nil {
		return
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return
	}

	// GPS coordinates.
	if lat, lon, err := x.LatLong(); err == nil {
		item.GPSLat = lat
		item.GPSLon = lon
		item.HasGPS = true
		loc := geo.FromGPS(lat, lon)
		item.Country = loc.Country
		item.CountryCode = loc.CountryCode
		item.Region = loc.Region
		item.Continent = loc.Continent
	}

	// Capture date.
	if t, err := x.DateTime(); err == nil {
		item.TakenAt = t
	}

	// Pixel dimensions.
	if tag, err := x.Get(exif.PixelXDimension); err == nil {
		if v, err := tag.Int(0); err == nil {
			item.Width = v
		}
	}
	if tag, err := x.Get(exif.PixelYDimension); err == nil {
		if v, err := tag.Int(0); err == nil {
			item.Height = v
		}
	}
}
