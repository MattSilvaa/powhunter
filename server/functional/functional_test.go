//go:build functional

// Package functional holds black-box tests that run against a deployed
// Powhunter: the API over HTTP and, optionally, the static web app.
//
// The same suite gates both sides of a deploy. CI runs it against a stack it
// starts itself, and the deploy workflow runs it against production after each
// release. Tests that write data run only when FUNCTIONAL_MAIL_LOG is set,
// which only a local stack can provide, so production only ever sees reads.
//
// Settings, all from the environment:
//
//	FUNCTIONAL_API_URL      required, e.g. https://api.powhunter.app
//	FUNCTIONAL_WEB_URL      optional; web checks are skipped without it
//	FUNCTIONAL_EXPECT_COMMIT optional; wait until /health reports this commit
//	FUNCTIONAL_MAIL_LOG     optional; the API's log file, which enables the
//	                        sign-up and sign-in flow (it reads the emailed link)
package functional

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	requestTimeout = 15 * time.Second
	// commitWait covers the gap between Render marking a deploy live and every
	// instance serving it.
	commitWait = 3 * time.Minute
)

var (
	apiURL      = strings.TrimRight(os.Getenv("FUNCTIONAL_API_URL"), "/")
	webURL      = strings.TrimRight(os.Getenv("FUNCTIONAL_WEB_URL"), "/")
	wantCommit  = os.Getenv("FUNCTIONAL_EXPECT_COMMIT")
	mailLogPath = os.Getenv("FUNCTIONAL_MAIL_LOG")
)

func TestMain(m *testing.M) {
	if apiURL == "" {
		fmt.Fprintln(os.Stderr, "FUNCTIONAL_API_URL is required")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// TestHealth runs first so the rest of the suite talks to the expected release.
func TestHealth(t *testing.T) {
	deadline := time.Now().Add(commitWait)

	for {
		var body struct {
			Status string `json:"status"`
			Commit string `json:"commit"`
		}

		status, err := getJSON(newClient(t), apiURL+"/health", &body)
		if err == nil && status == http.StatusOK && body.Status == "ok" &&
			(wantCommit == "" || body.Commit == wantCommit) {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("/health never reported commit %q: status=%d body=%+v err=%v",
				wantCommit, status, body, err)
		}

		time.Sleep(5 * time.Second)
	}
}

func TestReady(t *testing.T) {
	var body map[string]string
	status, err := getJSON(newClient(t), apiURL+"/ready", &body)
	if err != nil || status != http.StatusOK {
		t.Fatalf("/ready: status=%d body=%v err=%v", status, body, err)
	}
}

func TestResorts(t *testing.T) {
	resorts := listResorts(t)
	if len(resorts) == 0 {
		t.Fatal("/api/resorts returned no resorts")
	}
}

func TestSnowReport(t *testing.T) {
	var body struct {
		Mode string `json:"mode"`
	}

	status, err := getJSON(newClient(t), apiURL+"/api/snow-report", &body)
	if err != nil || status != http.StatusOK || body.Mode == "" {
		t.Fatalf("/api/snow-report: status=%d mode=%q err=%v", status, body.Mode, err)
	}
}

func TestSecurityHeaders(t *testing.T) {
	resp := do(t, newClient(t), http.MethodGet, apiURL+"/api/resorts", nil)
	resp.Body.Close()

	for _, header := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy"} {
		if resp.Header.Get(header) == "" {
			t.Errorf("missing %s", header)
		}
	}
}

func TestSignedOutIsRejected(t *testing.T) {
	resp := do(t, newClient(t), http.MethodGet, apiURL+"/api/auth/me", nil)
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/api/auth/me signed out: status=%d, want 401", resp.StatusCode)
	}
}

// TestWebApp checks the pages and that the scripts they load exist, which
// catches a build that produced HTML pointing at missing bundles.
func TestWebApp(t *testing.T) {
	if webURL == "" {
		t.Skip("FUNCTIONAL_WEB_URL not set")
	}

	client := newClient(t)
	home := getText(t, client, webURL+"/")

	if !strings.Contains(home, "<title>Pow Hunter") {
		t.Error("home page has no Pow Hunter title")
	}

	scripts := regexp.MustCompile(`"(/assets/[^"]+\.js)"`).FindAllStringSubmatch(home, -1)
	if len(scripts) == 0 {
		t.Fatal("home page references no scripts")
	}

	for _, script := range scripts {
		getText(t, client, webURL+script[1])
	}

	for _, path := range []string{"/resorts", "/signup", "/sitemap.xml", "/robots.txt"} {
		getText(t, client, webURL+path)
	}
}

// TestSignUpAndManageAlerts walks the main user journey: sign up, sign in
// with the emailed link, see the alert, delete it and sign out.
func TestSignUpAndManageAlerts(t *testing.T) {
	if mailLogPath == "" {
		t.Skip("FUNCTIONAL_MAIL_LOG not set: write tests only run against a local stack")
	}

	client := newClient(t)
	resort := listResorts(t)[0]
	email := fmt.Sprintf("functional+%d@example.com", time.Now().UnixNano())

	expectStatus(t, do(t, client, http.MethodPost, apiURL+"/api/alerts", map[string]any{
		"email":            email,
		"phone":            "415-555-0123",
		"notificationDays": 3,
		"minSnowAmount":    6,
		"resortsUuids":     []string{resort.UUID},
	}), http.StatusOK, http.StatusCreated)

	logOffset := mailLogSize(t)
	expectStatus(t, do(t, client, http.MethodPost, apiURL+"/api/auth/request-link",
		map[string]string{"email": email}), http.StatusOK)

	token := waitForLoginToken(t, logOffset)
	expectStatus(t, do(t, client, http.MethodPost, apiURL+"/api/auth/callback",
		map[string]string{"token": token}), http.StatusOK)

	if alerts := userAlerts(t, client); !strings.Contains(alerts, resort.Name) {
		t.Fatalf("signed-in alerts do not include %s: %s", resort.Name, alerts)
	}

	expectStatus(t, do(t, client, http.MethodDelete, apiURL+"/api/user/alerts/delete-all", nil),
		http.StatusOK, http.StatusNoContent)

	if alerts := userAlerts(t, client); strings.Contains(alerts, resort.Name) {
		t.Fatalf("alerts still present after delete-all: %s", alerts)
	}

	expectStatus(t, do(t, client, http.MethodPost, apiURL+"/api/auth/logout", nil),
		http.StatusOK, http.StatusNoContent)
	expectStatus(t, do(t, client, http.MethodGet, apiURL+"/api/auth/me", nil), http.StatusUnauthorized)
}

type resort struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

func listResorts(t *testing.T) []resort {
	t.Helper()

	var resorts []resort
	status, err := getJSON(newClient(t), apiURL+"/api/resorts", &resorts)
	if err != nil || status != http.StatusOK {
		t.Fatalf("/api/resorts: status=%d err=%v", status, err)
	}

	return resorts
}

func userAlerts(t *testing.T, client *http.Client) string {
	t.Helper()

	resp := do(t, client, http.MethodGet, apiURL+"/api/user/alerts", nil)
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/user/alerts: status=%d body=%s", resp.StatusCode, body)
	}

	return body
}

var loginLink = regexp.MustCompile(`/login\?token=([^"\\\s&<]+)`)

// waitForLoginToken reads the sign-in link the API logged in place of sending
// an email, looking only at log lines written after offset.
func waitForLoginToken(t *testing.T, offset int64) string {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		data, err := os.ReadFile(mailLogPath)
		if err != nil {
			t.Fatalf("read mail log: %v", err)
		}

		if matches := loginLink.FindAllSubmatch(data[offset:], -1); len(matches) > 0 {
			token, err := url.QueryUnescape(string(matches[len(matches)-1][1]))
			if err != nil {
				t.Fatalf("unescape token: %v", err)
			}

			return token
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Fatal("no sign-in link appeared in the mail log")

	return ""
}

func mailLogSize(t *testing.T) int64 {
	t.Helper()

	info, err := os.Stat(mailLogPath)
	if err != nil {
		t.Fatalf("stat mail log: %v", err)
	}

	return info.Size()
}

func newClient(t *testing.T) *http.Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}

	return &http.Client{Timeout: requestTimeout, Jar: jar}
}

func do(t *testing.T, client *http.Client, method, target string, body any) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode body: %v", err)
		}

		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, target, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}

	return resp
}

func getJSON(client *http.Client, target string, into any) (int, error) {
	resp, err := client.Get(target)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, json.NewDecoder(resp.Body).Decode(into)
}

func getText(t *testing.T, client *http.Client, target string) string {
	t.Helper()

	resp := do(t, client, http.MethodGet, target, nil)
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status=%d", target, resp.StatusCode)
	}

	return body
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return string(data)
}

func expectStatus(t *testing.T, resp *http.Response, want ...int) {
	t.Helper()

	body := readBody(t, resp)
	for _, status := range want {
		if resp.StatusCode == status {
			return
		}
	}

	t.Fatalf("%s %s: status=%d, want %v; body=%s",
		resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, want, body)
}
