package main

import (
	"fmt"
	"strings"
)

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

// Spinner frames for animated loading indicator
var Spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ASCII art banner
const banner = `
%s██╗   ██╗████████╗    ███████╗███╗   ██╗██╗██████╗ ███████╗██████╗ %s
%s██║   ██║╚══██╔══╝    ██╔════╝████╗  ██║██║██╔══██╗██╔════╝██╔══██╗%s
%s██║   ██║   ██║       ███████╗██╔██╗ ██║██║██████╔╝█████╗  ██████╔╝%s
%s╚██╗ ██╔╝   ██║       ╚════██║██║╚██╗██║██║██╔═══╝ ██╔══╝  ██╔══██╗%s
%s ╚████╔╝    ██║       ███████║██║ ╚████║██║██║     ███████╗██║  ██║%s
%s  ╚═══╝     ╚═╝       ╚══════╝╚═╝  ╚═══╝╚═╝╚═╝     ╚══════╝╚═╝  ╚═╝%s
`

// Box drawing width
const boxWidth = 50

// PrintBanner displays the ASCII art banner with gradient colors
func PrintBanner() {
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

// Box drawing helpers (open-right style to avoid alignment issues with variable-width icons)

func boxTop(color string) string {
	return fmt.Sprintf("%s╭%s%s", color, strings.Repeat("─", boxWidth), Reset)
}

func boxBottom(color string) string {
	return fmt.Sprintf("%s╰%s%s", color, strings.Repeat("─", boxWidth), Reset)
}

func boxLine(color string, content string) string {
	return fmt.Sprintf("%s│%s %s", color, Reset, content)
}

// PrintConfigBox displays the configuration summary in a styled box
func PrintConfigBox(crnCount int, email string, interval int, term string) {
	fmt.Println(boxTop(Dim))
	fmt.Println(boxLine(Dim, fmt.Sprintf("%s%s  Monitoring %s%d CRNs%s", Cyan, IconTarget, BoldWhite, crnCount, Reset)))
	if email != "" {
		fmt.Println(boxLine(Dim, fmt.Sprintf("%s%s  %s%s%s", Magenta, IconEmail, White, truncateString(email, 35), Reset)))
	}
	fmt.Println(boxLine(Dim, fmt.Sprintf("%s%s  Interval: %s%ds%s  %s%s  Term: %s%s%s", Yellow, IconClock, BoldWhite, interval, Reset, Cyan, IconCalendar, BoldWhite, term, Reset)))
	fmt.Println(boxBottom(Dim))
	fmt.Println()
}

// PrintFetchingHeader displays the "Fetching course information" message
func PrintFetchingHeader() {
	fmt.Printf("%s%s  Fetching course information...%s\n\n", Dim, IconSearch, Reset)
}

// PrintCourseFound displays a successfully found course
func PrintCourseFound(crn, name string) {
	fmt.Printf("  %s%s%s %s%s%s %s▸%s %s\n", Green, IconCheck, Reset, Cyan, crn, Reset, Dim, Reset, name)
}

// PrintCourseNotFound displays a course that wasn't found
func PrintCourseNotFound(crn string) {
	fmt.Printf("  %s%s%s %s%s%s: %snot found, skipping%s\n", Red, IconX, Reset, Dim, crn, Reset, Red, Reset)
}

// PrintDivider displays a horizontal divider line
func PrintDivider() {
	fmt.Printf("\n%s────────────────────────────────────────────────────%s\n\n", Dim, Reset)
}

// PrintCheckingStatus displays the current checking status with spinner
func PrintCheckingStatus(spinnerIdx, attempt int, crn string) {
	fmt.Printf("\r%s%s%s %sAttempt #%d%s %s│%s Checking %s%s%s...                              ",
		Cyan, Spinner[spinnerIdx%len(Spinner)], Reset, Bold, attempt, Reset, Dim, Reset, Cyan, crn, Reset)
}

// PrintCheckError displays an error that occurred while checking a CRN
func PrintCheckError(checkTime, crn string, err error) {
	fmt.Printf("\r%s%s%s %s[%s]%s Error checking %s: %v\n",
		Red, IconX, Reset, Dim, checkTime, Reset, crn, err)
}

// PrintSeatAvailable displays the seat available success box
func PrintSeatAvailable(name, crn string) {
	ClearLine()
	fmt.Println()
	fmt.Println(boxTop(Green))
	fmt.Println(boxLine(Green, fmt.Sprintf("%s%s  SEAT AVAILABLE!%s", BoldGreen, IconCheck, Reset)))
	fmt.Println(boxLine(Green, fmt.Sprintf("  %s%s%s", White, name, Reset)))
	fmt.Println(boxLine(Green, fmt.Sprintf("  %sCRN: %s%s", Dim, crn, Reset)))
	fmt.Println(boxBottom(Green))
}

// PrintEmailSent displays the email notification confirmation
func PrintEmailSent(email string) {
	fmt.Printf("  %s%s%s %sNotification sent to %s%s\n\n", Magenta, IconEmail, Reset, Dim, email, Reset)
}

// PrintWaitingStatus displays the waiting status with spinner
func PrintWaitingStatus(spinnerIdx, attempt, found, total int, timeLeft, checkTime string) {
	fmt.Printf("\r%s%s%s %sAttempt #%d%s %s│%s Found: %s%d%s/%s%d%s %s│%s Next: %s%s%s %s[%s]%s          ",
		Cyan, Spinner[spinnerIdx%len(Spinner)], Reset,
		Bold, attempt, Reset,
		Dim, Reset,
		Green, found, Reset,
		Dim, total, Reset,
		Dim, Reset,
		Yellow, timeLeft, Reset,
		Dim, checkTime, Reset)
}

// PrintAllCoursesFound displays the completion message
func PrintAllCoursesFound() {
	fmt.Printf("\n%s%s  All courses found! Exiting...%s\n", BoldGreen, IconCheck, Reset)
}

// ClearLine clears the current terminal line
func ClearLine() {
	fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
}

// truncateString truncates a string to maxLen, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
