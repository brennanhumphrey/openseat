package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/resend/resend-go/v2"
)

const timetableUrl = "https://selfservice.banner.vt.edu/ssb/HZSKVTSC.P_ProcRequest"

type Config struct {
	CRN           string
	CheckInterval time.Duration
	Campus        string
	Term          string
	Email         string
}

func parseFlags() Config {
	crnPtr := flag.String("crn", "", "The CRN of the course section to monitor (required)")
	waitPtr := flag.Int("wait", 30, "Seconds to wait between checks")
	emailPtr := flag.String("email", "", "Email for notification")
	flag.Parse()

	if *crnPtr == "" {
		log.Fatal("Error: -crn flag is required")
	}

	return Config{
		CRN:           *crnPtr,
		CheckInterval: time.Duration(*waitPtr) * time.Second,
		Campus:        "0",      // Blacksburg
		Term:          "202601", // Spring 2026
		Email:         *emailPtr,
	}
}

func (c Config) buildPayload(openOnly bool) url.Values {
	// Initialize as a standard Go map
	rawMap := map[string][]string{
		"CAMPUS":           {c.Campus},
		"TERMYEAR":         {c.Term},
		"CORE_CODE":        {"AR%"},
		"subj_code":        {"%"},
		"SCHDTYPE":         {"%"},
		"CRSE_NUMBER":      {""},
		"crn":              {c.CRN},
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

func checkSectionOpen(cfg Config) (bool, error) {
	payload := cfg.buildPayload(true)
	doc, err := fetchDocument(timetableUrl, payload)
	if err != nil {
		return false, err
	}

	table := doc.Find(".dataentrytable").Text()
	return strings.Contains(table, cfg.CRN), nil
}

func getCourseName(cfg Config) (string, error) {
	payload := cfg.buildPayload(false)
	doc, err := fetchDocument(timetableUrl, payload)
	if err != nil {
		return "", err
	}

	var courseName string
	doc.Find(".dataentrytable tr").Each(func(i int, row *goquery.Selection) {
		// check if the row contains the target crn
		if strings.Contains(row.Find("td:nth-child(1)").Text(), cfg.CRN) {
			// the course title is in the 3rd td cell
			courseName = strings.TrimSpace(row.Find("td:nth-child(3)").Text())
		}
	})

	if courseName == "" {
		return "", fmt.Errorf("course not found for CRN: %s", cfg.CRN)
	}

	return courseName, nil
}

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

func main() {
	cfg := parseFlags()

	courseName, err := getCourseName(cfg)
	if err != nil {
		log.Fatalf("Failed to get course name : %v", err)
	}
	fmt.Printf("Monitoring: %s (CRN: %s)\n\n", courseName, cfg.CRN)

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	for attempt := 1; ; attempt++ {
		// Animate spinner while waiting
		fmt.Printf("\r%s [Attempt %d] Checking...                    ", spinner[attempt%len(spinner)], attempt)

		open, err := checkSectionOpen(cfg)
		checkTime := time.Now().Format("15:04:05")

		if err != nil {
			fmt.Printf("\r✗ [%s] Error: %v                           ", checkTime, err)
		} else if open {
			msg := fmt.Sprintf("OPEN SEAT in %s (CRN: %s)!", courseName, cfg.CRN)
			fmt.Printf("\r\n%s\n", msg)
			// TODO: add notification (email, sms, etc.)

			if cfg.Email != "" {
				fmt.Println("Sending email notification...")
				err := sendEmail(cfg.Email, "VT Course Section Open!", msg)
				if err != nil {
					fmt.Printf("Failed to send email: %v\n", err)
				} else {
					fmt.Printf("Email sent to %s\n", cfg.Email)
				}
			}

			return
		}

		// Animate the spinner while waiting
		waitUntil := time.Now().Add(cfg.CheckInterval)
		i := 0
		for time.Now().Before(waitUntil) {
			remaining := time.Until(waitUntil).Round(time.Second)
			fmt.Printf("\r%s [%s] Attempt %d - Not open. Next check in %v...     ",
				spinner[i%len(spinner)], checkTime, attempt, remaining)
			time.Sleep(100 * time.Millisecond)
			i++
		}
	}
}
