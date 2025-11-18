package main

import (
	"log"
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Startup struct {
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	Tagline             string    `json:"tagline"`
	Description         string    `json:"description"`
	BannerURL           string    `json:"banner_url"`
	LogoURL             string    `json:"logo_url"`
	WebsiteURL          string    `json:"website_url"`
	JobsURL             string    `json:"jobs_url"`
	Location            string    `json:"location"`
	FundingStage        string    `json:"funding_stage"`
	Industry            string    `json:"industry"`
	WorkType            string    `json:"work_type"`
	TeamSize            string    `json:"team_size"`
	FundingAnnouncement string    `json:"funding_announcement,omitempty"`
	LastFetched         time.Time `json:"last_fetched"`
	FullyScraped        bool      `json:"fully_scraped"`
}

type startupsLoadedMsg struct {
	startups []*Startup
}
type CompanyPreview struct {
	Slug    string
	Name    string
	Tagline string
}

var (
	bloombergOrange = lipgloss.Color("#FF6B35")
	bloombergBlue   = lipgloss.Color("#004E89")
	bloombergGreen  = lipgloss.Color("#00D9FF")
	bloombergYellow = lipgloss.Color("#F6AE2D")
	bloombergBg     = lipgloss.Color("#000000")
	bloombergText   = lipgloss.Color("#FFFFFF")
	bloombergGray   = lipgloss.Color("#808080")
)

func openURL(url string) {
	if url == "" {
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		log.Printf("Unsupported platform: %s", runtime.GOOS)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open URL: %v", err)
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
