package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const timetableUrl = "https://selfservice.banner.vt.edu/ssb/HZSKVTSC.P_ProcRequest"

type Config struct {
	CRN           string
	CheckInterval time.Duration
	Campus        string
	Term          string
}

func parseFlags() Config {
	crnPtr := flag.String("crn", "", "The CRN of the course section to monitor (required)")
	waitPtr := flag.Int("wait", 30, "Seconds to wait between checks")

	flag.Parse()

	if *crnPtr == "" {
		log.Fatal("Error: -crn flag is required")
	}

	return Config{
		CRN:           *crnPtr,
		CheckInterval: time.Duration(*waitPtr) * time.Second,
		Campus:        "0",      // Blacksburg
		Term:          "202601", // Spring 2026
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

func main() {
	targetCrn := *crnPtr
	refreshSeconds := time.Duration(*waitPtr)

	// payload := url.Values{
	// 	"CAMPUS":           {"0"},      // Blacksburg
	// 	"TERMYEAR":         {"202601"}, // Spring 2026
	// 	"CORE_CODE":        {"AR%"},
	// 	"subj_code":        {"%"},
	// 	"SCHDTYPE":         {"%"},
	// 	"CRSE_NUMBER":      {""},
	// 	"crn":              {TARGET_CRN},
	// 	"open_only":        {"on"}, // only result if section is open/not full
	// 	"sess_code":        {"%"},
	// 	"BTN_PRESSED":      {"FIND class sections"},
	// 	"inst_name":        {""},
	// 	"disp_comments_in": {""},
	// }

	// Initialize as a standard Go map
	rawMap := map[string][]string{
		"CAMPUS":           {"0"},      // Blacksburg
		"TERMYEAR":         {"202601"}, // Spring 2026
		"CORE_CODE":        {"AR%"},
		"subj_code":        {"%"},
		"SCHDTYPE":         {"%"},
		"CRSE_NUMBER":      {""},
		"crn":              {targetCrn},
		"open_only":        {"on"}, // result only if section is open
		"sess_code":        {"%"},
		"BTN_PRESSED":      {"FIND class sections"},
		"inst_name":        {""},
		"disp_comments_in": {""},
	}
	// Convert the map to the url.Values type so it can be passed into http methods
	payload := url.Values(rawMap)

	found := false
	attempt := 1
	for !found {
		log.Printf("Checking for opening in CRN: %s (Attempt %d)...\n", targetCrn, attempt)

		resp, err := http.PostForm(URL, payload)
		if err != nil {
			log.Fatal(err)
		}

		if resp.StatusCode != 200 {
			log.Fatalf("status code error: %d %s", resp.StatusCode, resp.Status)
		}

		// Load the HTML document
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close() // close early to avoid memory leak

		// this is the html table that would contain the info for a matching sections
		table := doc.Find(".dataentrytable").Text()

		if strings.Contains(table, targetCrn) {
			found = true
		}

		if !found {
			log.Printf("Not open yet. Waiting %d seconds before trying again.", refreshSeconds)
		}
		time.Sleep(refreshSeconds * time.Second)

		attempt++
	}
}

func getCourseName(crn string, timetableUrl string, payload url.Values) (string, error) {
	payload.Set("open_only", "")

	doc, err := sendPostRequest(timetableUrl, payload)
	if err != nil {
		log.Fatal(err)
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

func sendPostRequest(timetableUrl string, payload url.Values) (*goquery.Document, error) {
	resp, err := http.PostForm(timetableUrl, payload)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return doc, err
}
