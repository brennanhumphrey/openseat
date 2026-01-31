# VT Sniper

A command-line tool for monitoring Virginia Tech course section availability and receiving instant email notifications when seats open up.

<p align="center">
    <img src="assets/demo.gif" width="600" alt="VT Sniper Demo">
</p>

## Overview

VT Sniper automates the tedious process of repeatedly checking the Virginia Tech course registration system for available seats. It monitors one or more Course Reference Numbers (CRNs) and sends you an email notification the moment a seat becomes available in your desired class.

## Features

- **Instant Email Notifications** - Get an email the moment a seat opens up in your desired course
- **Multi-Course Monitoring** - Track multiple CRNs simultaneously with a single command
- **Free & Open Source** - No subscriptions, no fees, no data collection
- **Runs Locally** - Your data stays on your machine; no third-party services watching your courses
- **Configurable Polling** - Adjustable check interval (default: 30 seconds)
- **Real-Time Status Display** - Beautiful terminal UI with live progress updates
- **Cross-Platform** - Works on macOS, Linux, and Windows

## Prerequisites

- Go 1.21 or later
- A [Resend](https://resend.com) account and API key (free tier available)
- A [Nerd Font](https://www.nerdfonts.com/) installed in your terminal (optional, for icons)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/brennanhumphrey/vt-sniper.git
cd vt-sniper

# Build the binary
go build -o vt-sniper

# Or install directly
go install
```

## Configuration

### 1. Create a Configuration File

Create a `config.json` file in the project directory:

```json
{
  "crns": ["12345", "67890", "11111"],
  "email": "your.email@vt.edu",
  "checkInterval": 30,
  "term": "202601",
  "campus": "0"
}
```

### Configuration Options

| Field           | Type     | Required | Default    | Description                                       |
| --------------- | -------- | -------- | ---------- | ------------------------------------------------- |
| `crns`          | string[] | Yes      | -          | List of Course Reference Numbers to monitor       |
| `email`         | string   | Yes      | -          | Email address for notifications                   |
| `checkInterval` | int      | No       | `30`       | Seconds between availability checks               |
| `term`          | string   | No       | `"202601"` | Academic term code (e.g., `202601` = Spring 2026) |
| `campus`        | string   | No       | `"0"`      | Campus code (`0` = Blacksburg)                    |

### Term Code Format

Term codes follow the pattern `YYYYMM`:

- `01` = Spring
- `06` = Summer I
- `07` = Summer II
- `09` = Fall

Examples:

- `202601` = Spring 2026
- `202509` = Fall 2025
- `202506` = Summer I 2025

### 2. Set Up Email Notifications

1. Create a free account at [Resend](https://resend.com) (free tier includes 100 emails/day and 3,000 emails/month)
2. Generate an API key from your Resend dashboard
3. Set the API key using one of these methods:

**Option A: Create a `.env` file** (recommended)

Create a `.env` file in the project directory:

```bash
RESEND_API_KEY=re_your_api_key_here
```

**Option B: Export as environment variable**

```bash
export RESEND_API_KEY="re_your_api_key_here"
```

For persistent configuration, add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
echo 'export RESEND_API_KEY="re_your_api_key_here"' >> ~/.zshrc
source ~/.zshrc
```

## Usage

```bash
# Run the monitor
./vt-sniper
```

### Example Output

The CLI features a stylized interface with ANSI colors, box drawing, and Nerd Font icons.

<!-- Add demo gif/video/images here -->

> **Note:** For the best visual experience, use a terminal with a [Nerd Font](https://www.nerdfonts.com/) installed (e.g., FiraCode Nerd Font, JetBrains Mono Nerd Font). The tool will still work without Nerd Fonts, but icons may not display correctly.

### Tips for Reliable Monitoring

To ensure VT Sniper runs continuously without interruption:

**Keep it running when you close the terminal:**

```bash
# Using nohup (output goes to nohup.out)
nohup ./vt-sniper &

# Using tmux (recommended - lets you reattach later)
tmux new -s sniper
./vt-sniper
# Press Ctrl+B, then D to detach
# Reattach later with: tmux attach -t sniper

# Using screen
screen -S sniper
./vt-sniper
# Press Ctrl+A, then D to detach
# Reattach later with: screen -r sniper
```

**Prevent your computer from sleeping:**

- **macOS:** Use `caffeinate -i ./vt-sniper` to prevent sleep while running
- **Linux:** Disable sleep in power settings or use `systemd-inhibit ./vt-sniper`
- **Windows:** Adjust power settings to prevent sleep when plugged in

**For 24/7 monitoring:**

- Run on a Raspberry Pi or old laptop that can stay on
- Use a cloud VM (AWS, DigitalOcean, etc.) - many have free tiers
- Use a home server if you have one

**Stopping background processes:**

> ⚠️ If you use `nohup` or detach from `tmux`/`screen`, the process keeps running in the background. Don't forget to stop it when you're done!

```bash
# Find the process
ps aux | grep vt-sniper

# Kill it by PID
kill <PID>

# Or kill all instances
pkill vt-sniper

# If using tmux
tmux attach -t sniper
# Then press Ctrl+C to stop, and type 'exit' to close the session

# If using screen
screen -r sniper
# Then press Ctrl+C to stop, and type 'exit' to close the session
```

## Finding CRNs

1. Go to [Virginia Tech Timetable](https://banweb.banner.vt.edu/ssb/prod/HZSKVTSC.P_DispRequest)
2. Search for your desired course
3. The CRN is displayed in the first column of the search results

## How It Works

1. **Configuration Loading** - Reads CRNs and settings from `config.json`
2. **CRN Validation** - Verifies each CRN exists and fetches course names
3. **Polling Loop** - Periodically checks Virginia Tech's course system for availability
4. **Notification** - Sends email via Resend API when a seat opens up
5. **Completion** - Exits when all monitored courses have available seats (or on interrupt)

The tool queries Virginia Tech's Banner self-service system and parses the HTML response to determine if seats are available in the "Open Sections Only" view.

## Development

### Project Structure

```
vt-sniper-go/
├── main.go           # Application entry point
├── vt_sniper.go      # Core monitoring logic
├── ui.go             # Terminal UI (colors, icons, formatting)
├── vt_sniper_test.go # Unit tests
├── config.json       # Configuration file (create this)
├── go.mod            # Go module definition
├── go.sum            # Dependency checksums
└── README.md         # This file
```

### Running Tests

```bash
# Run all tests
go test -v

# Run with coverage
go test -cover

# Generate coverage report
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Dependencies

| Package                                           | Purpose                            |
| ------------------------------------------------- | ---------------------------------- |
| [goquery](https://github.com/PuerkitoBio/goquery) | HTML parsing and DOM traversal     |
| [resend-go](https://github.com/resend/resend-go)  | Email notifications via Resend API |

## Troubleshooting

### "RESEND_API_KEY not set"

Ensure the environment variable is exported in your current shell session:

```bash
export RESEND_API_KEY="re_your_api_key_here"
```

### "Failed to load config"

Verify that `config.json` exists in the current directory and contains valid JSON with at least one CRN.

### "CRN not found"

The CRN may be invalid for the specified term. Double-check the CRN on the [VT Timetable](https://banweb.banner.vt.edu/ssb/prod/HZSKVTSC.P_DispRequest).

### Rate Limiting

The tool includes a 500ms delay between individual course checks to avoid overwhelming Virginia Tech's servers. If you experience connection issues, try increasing `checkInterval` in your configuration.

## Disclaimer

This tool is intended for personal use to assist with course registration. Please use responsibly and in accordance with Virginia Tech's acceptable use policies. The author is not responsible for any misuse of this tool.

## License

MIT License - See [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
