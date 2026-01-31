// Package main implements a CLI tool for monitoring Virginia Tech course sections
// and notifying users when seats become available.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/resend/resend-go/v2"
)

// DefaultTimetableURL is the Virginia Tech timetable endpoint for course searches
const DefaultTimetableURL = "https://selfservice.banner.vt.edu/ssb/HZSKVTSC.P_ProcRequest"

// ANSI color codes
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Dim        = "\033[2m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Magenta    = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	BoldGreen  = "\033[1;32m"
	BoldCyan   = "\033[1;36m"
	BoldYellow = "\033[1;33m"
	BoldRed    = "\033[1;31m"
	BoldWhite  = "\033[1;37m"
)

// Nerd Font icons (requires a Nerd Font to display correctly)
const (
	IconSearch   = "\uf002" //  (nf-fa-search)
	IconEmail    = "\uf0e0" //  (nf-fa-envelope)
	IconClock    = "\uf017" //  (nf-fa-clock)
	IconCheck    = "\uf00c" //  (nf-fa-check)
	IconX        = "\uf00d" //  (nf-fa-times)
	IconBook     = "\uf02d" //  (nf-fa-book)
	IconTarget   = "\uf140" //  (nf-fa-crosshairs)
	IconBell     = "\uf0f3" //  (nf-fa-bell)
	IconArrow    = "\uf061" //  (nf-fa-arrow_right)
	IconCalendar = "\uf073" //  (nf-fa-calendar)
	IconGrad     = "\uf19d" //  (nf-fa-graduation_cap)
)

// ASCII art banner
const banner = `
%s██╗   ██╗████████╗    ███████╗███╗   ██╗██╗██████╗ ███████╗██████╗ %s
%s██║   ██║╚══██╔══╝    ██╔════╝████╗  ██║██║██╔══██╗██╔════╝██╔══██╗%s
%s██║   ██║   ██║       ███████╗██╔██╗ ██║██║██████╔╝█████╗  ██████╔╝%s
%s╚██╗ ██╔╝   ██║       ╚════██║██║╚██╗██║██║██╔═══╝ ██╔══╝  ██╔══██╗%s
%s ╚████╔╝    ██║       ███████║██║ ╚████║██║██║     ███████╗██║  ██║%s
%s  ╚═══╝     ╚═╝       ╚══════╝╚═╝  ╚═══╝╚═╝╚═╝     ╚══════╝╚═╝  ╚═╝%s
`

func printBanner() {
	fmt.Printf(banner,
		BoldCyan, Reset,
		BoldCyan, Reset,
		Cyan, Reset,
		Cyan, Reset,
		Blue, Reset,
		Blue, Reset,
	)
	fmt.Printf("%s%s  Virginia Tech Course Availability Monitor%s\n\n", Dim, IconGrad, Reset)
}

func truncateEmail(email string, maxLen int) string {
	if len(email) <= maxLen {
		return email
	}
	return email[:maxLen-3] + "..."
}

// Box drawing helpers (open-right style to avoid alignment issues with variable-width icons)
const boxWidth = 50

func boxTop(color string) string {
	return fmt.Sprintf("%s╭%s%s", color, strings.Repeat("─", boxWidth), Reset)
}

func boxBottom(color string) string {
	return fmt.Sprintf("%s╰%s%s", color, strings.Repeat("─", boxWidth), Reset)
}

func boxLine(color string, content string) string {
	return fmt.Sprintf("%s│%s %s", color, Reset, content)
}

// ===================================
// Interfaces for dependency injection
// ===================================

// EmailSender abstracts email sending for testability
type EmailSender interface {
	Send(to, subject, body string) error
}

// ResendEmailSender is the production implementation using Resend API
type ResendEmailSender struct {
	APIKey string
}

func (r *ResendEmailSender) Send(to, subject, body string) error {
	if r.APIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(r.APIKey)
	params := &resend.SendEmailRequest{
		From:    "onboarding@resend.dev",
		To:      []string{to},
		Subject: subject,
		Text:    body,
	}

	_, err := client.Emails.Send(params)
	return err
}

// ==================================
// Configuration
// ==================================

// Config holds the runtime configuration for the course monitor
type Config struct {
	CRNs          []string `json:"crns"`          // Course Reference Number(s) to monitor
	Email         string   `json:"email"`         // Email address for notifications (optional)
	CheckInterval int      `json:"checkInterval"` // Time between availability checks
	Term          string   `json:"term"`          // Term code (e.g., 202601 = Spring 2026)
	Campus        string   `json:"campus"`        // Campus code (0 = Blacksburg)
	BaseURL       string   `json:"baseUrl"`       // Timetable URL (optional, for testability) (defaults to timetable url)
}

type CourseStatus struct {
	CRN   string
	Name  string
	Found bool
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	// set defaults
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 30
	}
	if cfg.Campus == "" {
		cfg.Campus = "0"
	}
	if cfg.Term == "" {
		cfg.Term = "202601"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultTimetableURL
	}

	if len(cfg.CRNs) == 0 {
		return Config{}, fmt.Errorf("no CRNs specified in config")
	}

	return cfg, nil
}

func (c Config) getBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultTimetableURL
}

// buildPayload constructs the form data for a timetable search request.
// If openOnly is true, results are filtered to sections with available seats.
func (c Config) buildPayload(crn string, openOnly bool) url.Values {
	// Initialize as a standard Go map
	rawMap := map[string][]string{
		"CAMPUS":           {c.Campus},
		"TERMYEAR":         {c.Term},
		"CORE_CODE":        {"AR%"},
		"subj_code":        {"%"},
		"SCHDTYPE":         {"%"},
		"CRSE_NUMBER":      {""},
		"crn":              {crn},
		"sess_code":        {"%"},
		"BTN_PRESSED":      {"FIND class sections"},
		"inst_name":        {""},
		"disp_comments_in": {""},
	}
	if openOnly {
		rawMap["open_only"] = []string{"on"}
	}
	// Convert the map to the url.Values type so it can be passed into http methods
	payload := url.Values(rawMap)

	return payload
}

// ====================================
// HTTP / Scraping
// ====================================

// fetchDocument sends a POST request to the given URL and parses the response as HTML.
// Returns the parsed document or an error if the request fails or returns non-200 status.
func fetchDocument(targetUrl string, payload url.Values) (*goquery.Document, error) {
	resp, err := http.PostForm(targetUrl, payload)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc, err
}

// checkSectionOpen checks if the configured course section has available seats.
// Returns true if the section appears in open-only search results.
func (c Config) checkSectionOpen(crn string) (bool, error) {
	payload := c.buildPayload(crn, true)
	doc, err := fetchDocument(c.getBaseURL(), payload)
	if err != nil {
		return false, err
	}

	table := doc.Find(".dataentrytable").Text()
	return strings.Contains(table, crn), nil
}

// getCourseName retrieves the course title for the configured CRN.
// Returns an error if the CRN is not found in the timetable.
func (c Config) getCourseName(crn string) (string, error) {
	payload := c.buildPayload(crn, false)
	doc, err := fetchDocument(c.BaseURL, payload)
	if err != nil {
		return "", err
	}

	var courseName string
	doc.Find(".dataentrytable tr").Each(func(i int, row *goquery.Selection) {
		// check if the row contains the target crn
		if strings.Contains(row.Find("td:nth-child(1)").Text(), crn) {
			// the course title is in the 3rd td cell
			courseName = strings.TrimSpace(row.Find("td:nth-child(3)").Text())
		}
	})

	if courseName == "" {
		return "", fmt.Errorf("course not found for CRN: %s", crn)
	}

	return courseName, nil
}

// =================================
// Notifications
// =================================

// sendEmail sends a notification email using the Resend API.
// Requires RESEND_API_KEY environment varialbe to be set.
func sendEmail(to, subject, body string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    "onboarding@resend.dev",
		To:      []string{to},
		Subject: subject,
		Text:    body,
		// Html: "<p>Hello, World!</p>",
	}

	_, err := client.Emails.Send(params)
	return err
}

// ===================================
// Main Function
// ===================================

type RunOptions struct {
	ConfigPath  string
	EmailSender EmailSender
}

func Run(opts RunOptions) error {
	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// use provided email sender or create default
	emailSender := opts.EmailSender
	if emailSender == nil {
		emailSender = &ResendEmailSender{APIKey: os.Getenv("RESEND_API_KEY")}
	}

	// Print banner
	printBanner()

	// Print configuration summary in a box
	fmt.Println(boxTop(Dim))
	fmt.Println(boxLine(Dim, fmt.Sprintf("%s%s  Monitoring %s%d CRNs%s", Cyan, IconTarget, BoldWhite, len(cfg.CRNs), Reset)))
	if cfg.Email != "" {
		fmt.Println(boxLine(Dim, fmt.Sprintf("%s%s  %s%s%s", Magenta, IconEmail, White, truncateEmail(cfg.Email, 35), Reset)))
	}
	fmt.Println(boxLine(Dim, fmt.Sprintf("%s%s  Interval: %s%ds%s  %s%s  Term: %s%s%s", Yellow, IconClock, BoldWhite, cfg.CheckInterval, Reset, Cyan, IconCalendar, BoldWhite, cfg.Term, Reset)))
	fmt.Println(boxBottom(Dim))
	fmt.Println()

	// initialize course statuses - filter out invalid CRNs
	fmt.Printf("%s%s  Fetching course information...%s\n\n", Dim, IconSearch, Reset)
	var courses []CourseStatus
	for _, crn := range cfg.CRNs {
		name, err := cfg.getCourseName(crn)
		if err != nil {
			fmt.Printf("  %s%s%s %s%s%s: %snot found, skipping%s\n", Red, IconX, Reset, Dim, crn, Reset, Red, Reset)
			continue
		}
		courses = append(courses, CourseStatus{CRN: crn, Name: name, Found: false})
		fmt.Printf("  %s%s%s %s%s%s %s▸%s %s\n", Green, IconCheck, Reset, Cyan, crn, Reset, Dim, Reset, name)
	}

	if len(courses) == 0 {
		return fmt.Errorf("no valid CRNs to monitor")
	}

	fmt.Printf("\n%s────────────────────────────────────────────────────%s\n\n", Dim, Reset)

	remaining := len(courses)
	interval := time.Duration(cfg.CheckInterval) * time.Second
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	for attempt := 1; ; attempt++ {
		checkTime := time.Now().Format("15:04:05")

		for i := range courses {
			if courses[i].Found {
				continue
			}

			fmt.Printf("\r%s%s%s %sAttempt #%d%s %s│%s Checking %s%s%s...                              ",
				Cyan, spinner[attempt%len(spinner)], Reset, Bold, attempt, Reset, Dim, Reset, Cyan, courses[i].CRN, Reset)

			open, err := cfg.checkSectionOpen(courses[i].CRN)
			if err != nil {
				fmt.Printf("\r%s%s%s %s[%s]%s Error checking %s: %v\n",
					Red, IconX, Reset, Dim, checkTime, Reset, courses[i].CRN, err)
				continue
			}

			if open {
				courses[i].Found = true
				remaining--

				// Clear line and print success
				fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
				fmt.Println()
				fmt.Println(boxTop(Green))
				fmt.Println(boxLine(Green, fmt.Sprintf("%s%s  SEAT AVAILABLE!%s", BoldGreen, IconCheck, Reset)))
				fmt.Println(boxLine(Green, fmt.Sprintf("  %s%s%s", White, courses[i].Name, Reset)))
				fmt.Println(boxLine(Green, fmt.Sprintf("  %sCRN: %s%s", Dim, courses[i].CRN, Reset)))
				fmt.Println(boxBottom(Green))

				if cfg.Email != "" {
					sendEmail(cfg.Email, "VT Course Section Open!", fmt.Sprintf("OPEN SEAT: %s (CRN: %s)", courses[i].Name, courses[i].CRN))
					fmt.Printf("  %s%s%s %sNotification sent to %s%s\n\n", Magenta, IconEmail, Reset, Dim, cfg.Email, Reset)
				}
			}

			time.Sleep(500 * time.Millisecond) // Small delay between requests
		}

		if remaining == 0 {
			fmt.Printf("\n%s%s  All courses found! Exiting...%s\n", BoldGreen, IconCheck, Reset)
			return nil
		}

		// Animate spinner while waiting
		waitUntil := time.Now().Add(interval)
		i := 0
		for time.Now().Before(waitUntil) {
			timeLeft := time.Until(waitUntil).Round(time.Second)
			found := len(courses) - remaining
			fmt.Printf("\r%s%s%s %sAttempt #%d%s %s│%s Found: %s%d%s/%s%d%s %s│%s Next: %s%v%s %s[%s]%s          ",
				Cyan, spinner[i%len(spinner)], Reset,
				Bold, attempt, Reset,
				Dim, Reset,
				Green, found, Reset,
				Dim, len(courses), Reset,
				Dim, Reset,
				Yellow, timeLeft, Reset,
				Dim, checkTime, Reset)
			time.Sleep(100 * time.Millisecond)
			i++
		}
	}
}
