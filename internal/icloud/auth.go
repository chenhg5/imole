package icloud

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

// endpoints for international and China iCloud.
type endpoints struct {
	auth  string // idmsa.*
	setup string // setup.*
}

var (
	intlEndpoints = endpoints{
		auth:  "https://idmsa.apple.com",
		setup: "https://setup.icloud.com",
	}
	cnEndpoints = endpoints{
		auth:  "https://idmsa.apple.com.cn",
		setup: "https://setup.icloud.com.cn",
	}
)

// Client is an authenticated iCloud HTTP client.
type Client struct {
	http     *http.Client
	jar      *cookiejar.Jar
	ep       endpoints
	domain   string // "com" or "cn"
	username string
	Session  *Session
}

// NewClient creates a Client for the given domain ("com" or "cn").
func NewClient(domain string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	ep := intlEndpoints
	if domain == "cn" {
		ep = cnEndpoints
	}
	return &Client{
		http:   &http.Client{Jar: jar},
		jar:    jar,
		ep:     ep,
		domain: domain,
	}, nil
}

// commonHeaders returns the Apple auth widget headers required for signin.
func (c *Client) commonHeaders() map[string]string {
	origin := "https://idmsa.apple.com"
	if c.domain == "cn" {
		origin = "https://idmsa.apple.com.cn"
	}
	return map[string]string{
		"Origin":                           origin,
		"Referer":                          origin + "/",
		"Content-Type":                     "application/json",
		"X-Apple-OAuth-Client-Id":          "d39ba9916b7251055b22c7f910e2ea796ee65e98b2ddecea8f5dde8d9d1a815d",
		"X-Apple-OAuth-Client-Type":        "firstPartyAuth",
		"X-Apple-OAuth-Redirect-URI":       "https://www.icloud.com",
		"X-Apple-OAuth-Require-Grant-Code": "true",
		"X-Apple-OAuth-Response-Mode":      "form_post",
		"X-Apple-OAuth-Response-Type":      "code",
		"X-Apple-OAuth-State":              "AUTH_STATE",
		"X-Apple-Widget-Key":               "d39ba9916b7251055b22c7f910e2ea796ee65e98b2ddecea8f5dde8d9d1a815d",
	}
}

func (c *Client) do(method, rawURL string, body interface{}, extraHeaders map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return c.http.Do(req)
}

func (c *Client) doJSON(method, rawURL string, body, out interface{}, extraHeaders map[string]string) (int, http.Header, error) {
	resp, err := c.do(method, rawURL, body, extraHeaders)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	if out != nil && len(data) > 0 {
		_ = json.Unmarshal(data, out)
	}
	return resp.StatusCode, resp.Header, nil
}

// Login authenticates with Apple ID and returns a ready Session.
// If a valid cached session exists it is returned immediately.
// interactive controls whether 2FA prompts are shown on stdin/stdout.
func Login(username, password, domain string, interactive bool, twoFAInput func() (string, error)) (*Client, error) {
	// Try cached session first.
	cached := LoadSession(username)
	if cached.IsValid() {
		c, err := NewClient(domain)
		if err != nil {
			return nil, err
		}
		c.username = username
		c.Session = cached
		// Inject cookies into jar.
		setupURL, _ := url.Parse(c.ep.setup)
		var httpCookies []*http.Cookie
		for _, sc := range cached.Cookies {
			httpCookies = append(httpCookies, &http.Cookie{Name: sc.Name, Value: sc.Value}) //nolint:gosec // G124: iCloud third-party cookie
		}
		c.jar.SetCookies(setupURL, httpCookies)
		return c, nil
	}

	c, err := NewClient(domain)
	if err != nil {
		return nil, err
	}
	c.username = username

	if err := c.srpLogin(username, password); err != nil {
		return nil, fmt.Errorf("SRP login failed: %w", err)
	}

	// Handle 2FA if required.
	if err := c.handle2FA(interactive, twoFAInput); err != nil {
		return nil, fmt.Errorf("2FA failed: %w", err)
	}

	// Complete setup and extract session.
	if err := c.accountLogin(); err != nil {
		return nil, fmt.Errorf("account login failed: %w", err)
	}

	if err := SaveSession(username, c.Session); err != nil {
		// Non-fatal: just means next run will re-authenticate.
		_ = err
	}
	return c, nil
}

// srpLogin performs the SRP exchange (init + complete).
func (c *Client) srpLogin(username, password string) error {
	srp, err := NewSRPClient()
	if err != nil {
		return err
	}
	headers := c.commonHeaders()

	// Step 1: init
	initBody := map[string]interface{}{
		"a":           srp.PublicB64(),
		"accountName": username,
		"protocols":   []string{"s2k", "s2k_fo"},
	}
	var initResp struct {
		Iteration int    `json:"iteration"`
		Salt      string `json:"salt"`
		Protocol  string `json:"protocol"`
		B         string `json:"b"`
		C         string `json:"c"`
	}
	status, respHeaders, err := c.doJSON("POST", c.ep.auth+"/appleauth/auth/signin/init", initBody, &initResp, headers)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("signin/init returned %d", status)
	}

	// Save session tracking headers.
	c.Session = &Session{Domain: c.domain}
	c.Session.Scnt = respHeaders.Get("scnt")
	c.Session.SessionID = respHeaders.Get("X-Apple-ID-Session-Id")

	// Step 2: compute SRP response.
	m1, _, err := srp.Respond(username, password, initResp.Salt, initResp.B, initResp.Protocol, initResp.Iteration)
	if err != nil {
		return err
	}

	// Step 3: complete
	completeHeaders := c.commonHeaders()
	if c.Session.Scnt != "" {
		completeHeaders["scnt"] = c.Session.Scnt
	}
	if c.Session.SessionID != "" {
		completeHeaders["X-Apple-ID-Session-Id"] = c.Session.SessionID
	}
	completeBody := map[string]interface{}{
		"accountName": username,
		"c":           initResp.C,
		"m1":          m1,
		"m2":          "", // we don't verify server proof
		"rememberMe":  true,
		"trustTokens": []string{},
	}
	_, completeHeaders2, err := c.doJSON("POST",
		c.ep.auth+"/appleauth/auth/signin/complete?isRememberMeEnabled=true",
		completeBody, nil, completeHeaders)
	if err != nil {
		return err
	}
	// Update session tracking headers.
	if v := completeHeaders2.Get("scnt"); v != "" {
		c.Session.Scnt = v
	}
	if v := completeHeaders2.Get("X-Apple-ID-Session-Id"); v != "" {
		c.Session.SessionID = v
	}
	if v := completeHeaders2.Get("X-Apple-Session-Token"); v != "" {
		c.Session.DSWebSessionToken = v
	}
	if v := completeHeaders2.Get("X-Apple-ID-Account-Country"); v != "" {
		c.Session.AccountCountry = v
	}
	return nil
}

// handle2FA checks whether 2FA is needed and if so either prompts (interactive)
// or returns an error.
func (c *Client) handle2FA(interactive bool, input func() (string, error)) error {
	// Check 2FA requirement by calling the auth endpoint.
	authHeaders := map[string]string{"Content-Type": "application/json"}
	if c.Session.Scnt != "" {
		authHeaders["scnt"] = c.Session.Scnt
	}
	if c.Session.SessionID != "" {
		authHeaders["X-Apple-ID-Session-Id"] = c.Session.SessionID
	}

	var authInfo struct {
		AuthType           string `json:"authType"`
		TrustedDeviceCount int    `json:"trustedDeviceCount"`
	}
	status, _, err := c.doJSON("GET", c.ep.auth+"/appleauth/auth", nil, &authInfo, authHeaders)
	if err != nil {
		return err
	}
	if status == 204 || authInfo.AuthType == "" {
		return nil // No 2FA needed.
	}

	// 2FA is required.
	if !interactive {
		return fmt.Errorf("two-factor authentication required; run imole icloud interactively to complete 2FA")
	}

	fmt.Print("Two-factor authentication code: ")
	code, err := input()
	if err != nil {
		return err
	}
	code = strings.TrimSpace(code)

	verifyHeaders := authHeaders
	verifyBody := map[string]interface{}{
		"securityCode": map[string]string{"code": code},
	}
	verifyStatus, _, err := c.doJSON("POST",
		c.ep.auth+"/appleauth/auth/verify/trusteddevice/securitycode",
		verifyBody, nil, verifyHeaders)
	if err != nil {
		return err
	}
	if verifyStatus != 204 && verifyStatus != 200 {
		return fmt.Errorf("2FA verification returned %d", verifyStatus)
	}

	// Trust this device.
	_, _, _ = c.doJSON("GET", c.ep.auth+"/appleauth/auth/2sv/trust", nil, nil, verifyHeaders)
	return nil
}

// accountLogin completes setup and extracts the usable session cookies.
func (c *Client) accountLogin() error {
	loginBody := map[string]interface{}{
		"accountCountryCode": c.Session.AccountCountry,
		"dsWebAuthToken":     c.Session.DSWebSessionToken,
		"extended_login":     true,
		"trustToken":         "",
	}
	resp, err := c.do("POST", c.ep.setup+"/setup/ws/1/accountLogin",
		loginBody, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("accountLogin returned %d: %s", resp.StatusCode, string(data))
	}

	var info struct {
		DsInfo struct {
			DSID string `json:"dsid"`
		} `json:"dsInfo"`
		UserPartition int `json:"userPartition"`
	}
	_ = json.Unmarshal(data, &info)

	c.Session.DSID = info.DsInfo.DSID
	c.Session.Partition = info.UserPartition

	// Collect all cookies from the jar for this endpoint.
	setupURL, _ := url.Parse(c.ep.setup)
	cookies := c.jar.Cookies(setupURL)
	c.Session.Cookies = cookiesToSimple(cookies)
	for _, ck := range cookies {
		switch ck.Name {
		case "X-APPLE-WEBAUTH-TOKEN":
			c.Session.WebAuthToken = ck.Value
		case "X-APPLE-WEBAUTH-USER":
			c.Session.WebAuthUser = ck.Value
		case "X-APPLE-WEBAUTH-VALIDATE":
			c.Session.WebAuthValidate = ck.Value
		case "X-APPLE-DS-WEB-SESSION-TOKEN":
			c.Session.DSWebSessionToken = ck.Value
		}
	}
	return nil
}

// StdinTwoFA reads a 2FA code from os.Stdin.  Use as twoFAInput in Login.
func StdinTwoFA() (string, error) {
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return "", fmt.Errorf("no input")
	}
	return sc.Text(), nil
}
