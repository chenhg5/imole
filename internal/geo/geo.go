// Package geo provides offline GPS-to-location reverse geocoding using embedded
// NaturalEarth country data. No network access required.
package geo

import (
	"sync"

	"github.com/sams96/rgeo"
)

// Location holds the human-readable location for a GPS coordinate.
type Location struct {
	Country     string // e.g. "China", "United States of America"
	CountryCode string // ISO 3166-1 alpha-2, e.g. "CN", "US"
	Continent   string // e.g. "Asia", "North America"
	Region      string // e.g. "Eastern Asia", "Northern America"
}

// cacheKey is a truncated (1-decimal) lat/lon pair to reduce cache entries.
type cacheKey struct{ lat, lon float64 }

var (
	once     sync.Once
	rgeoInst *rgeo.Rgeo
	initErr  error

	mu    sync.Mutex
	cache = make(map[cacheKey]Location)
)

func init_() {
	rgeoInst, initErr = rgeo.New(rgeo.Countries10)
}

func getInstance() (*rgeo.Rgeo, error) {
	once.Do(init_)
	return rgeoInst, initErr
}

// FromGPS converts GPS coordinates to a Location.
// Returns zero Location if coordinates are (0,0), invalid, or over ocean.
func FromGPS(lat, lon float64) Location {
	if lat == 0 && lon == 0 {
		return Location{}
	}

	// Round to 1 decimal (~11km precision) for cache efficiency.
	key := cacheKey{
		lat: float64(int(lat*10)) / 10,
		lon: float64(int(lon*10)) / 10,
	}

	mu.Lock()
	if loc, ok := cache[key]; ok {
		mu.Unlock()
		return loc
	}
	mu.Unlock()

	rg, err := getInstance()
	if err != nil {
		return Location{}
	}

	loc, err := rg.ReverseGeocode([]float64{lon, lat}) // rgeo takes [lon, lat]
	if err != nil {
		mu.Lock()
		cache[key] = Location{} // cache miss (ocean/uninhabited)
		mu.Unlock()
		return Location{}
	}

	result := Location{
		Country:     loc.Country,
		CountryCode: loc.CountryCode2,
		Continent:   loc.Continent,
		Region:      loc.Region,
	}

	mu.Lock()
	cache[key] = result
	mu.Unlock()
	return result
}

// MatchCountry returns true if the location matches the query string.
// Matches country name (case-insensitive partial) or ISO alpha-2 code.
// e.g. "china", "CN", "中国" (country name in English only for now).
func MatchCountry(loc Location, query string) bool {
	if loc.Country == "" {
		return false
	}
	q := toLower(query)
	return contains(toLower(loc.Country), q) ||
		toLower(loc.CountryCode) == q ||
		contains(toLower(loc.Continent), q) ||
		contains(toLower(loc.Region), q)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
