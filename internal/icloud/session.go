package icloud

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Session holds all state needed to make authenticated iCloud API calls.
type Session struct {
	AccountCountry string `json:"account_country"`
	DSID           string `json:"dsid"`
	// Session cookies
	WebAuthToken      string `json:"web_auth_token"`
	WebAuthUser       string `json:"web_auth_user"`
	WebAuthValidate   string `json:"web_auth_validate"`
	WebKB             string `json:"web_kb,omitempty"`
	DSWebSessionToken string `json:"ds_web_session_token,omitempty"`
	// Routing
	Domain    string `json:"domain"` // "com" or "cn"
	Partition int    `json:"partition"`
	// Timestamps
	SavedAt time.Time `json:"saved_at"`
	// Raw cookies for direct HTTP use
	Cookies []*SimpleCookie `json:"cookies,omitempty"`
	// Headers needed for re-authentication
	Scnt      string `json:"scnt,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// SimpleCookie is a serialisable subset of http.Cookie.
type SimpleCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// IsValid returns true if the session was saved recently enough to try.
// iCloud sessions last ~2 weeks; we conservatively accept up to 12 days.
func (s *Session) IsValid() bool {
	return s != nil &&
		s.WebAuthToken != "" &&
		time.Since(s.SavedAt) < 12*24*time.Hour
}

// Apply injects the session cookies into an http.Request.
func (s *Session) Apply(req *http.Request) {
	for _, c := range s.Cookies {
		req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value}) //nolint:gosec // G124: iCloud third-party cookie
	}
}

// sessionDir returns (and creates) the imole config directory.
func sessionDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".imole")
	return dir, os.MkdirAll(dir, 0o700)
}

func sessionPath(username string) (string, error) {
	dir, err := sessionDir()
	if err != nil {
		return "", err
	}
	// Sanitise username for use as filename.
	safe := ""
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			safe += string(r)
		} else {
			safe += "_"
		}
	}
	return filepath.Join(dir, "icloud-session-"+safe+".json"), nil
}

// SaveSession persists the session to disk.
func SaveSession(username string, s *Session) error {
	s.SavedAt = time.Now()
	p, err := sessionPath(username)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// LoadSession loads a previously saved session.  Returns nil if none exists.
func LoadSession(username string) *Session {
	p, err := sessionPath(username)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil
	}
	return &s
}

// DeleteSession removes the saved session (e.g. after auth failure).
func DeleteSession(username string) {
	p, _ := sessionPath(username)
	_ = os.Remove(p)
}

// cookiesFromJar extracts all cookies from an http.CookieJar for a given URL.
func cookiesToSimple(cookies []*http.Cookie) []*SimpleCookie {
	out := make([]*SimpleCookie, 0, len(cookies))
	for _, c := range cookies {
		out = append(out, &SimpleCookie{Name: c.Name, Value: c.Value})
	}
	return out
}
