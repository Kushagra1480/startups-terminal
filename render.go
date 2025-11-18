package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chromedp/chromedp"
)

type model struct {
	startups      []*Startup
	selectedIndex int
	width         int
	height        int
	loading       bool
	spinner       spinner.Model
	filterMode    bool
	filterInput   string
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(bloombergOrange)
	return model{
		startups: []*Startup{},
		loading:  true,
		spinner:  s,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadStartups,
	)
}
func loadStartups() tea.Msg {
	cache, err := LoadCache()
	if err == nil && time.Since(cache.LastUpdated) < 24*time.Hour {
		return startupsLoadedMsg{startups: cache.Startups}
	}
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	previews, err := ScrapeHomepage(ctx)
	if err != nil {
		log.Printf("Error scraping homepage: %v", err)
		return startupsLoadedMsg{startups: []*Startup{}}
	}

	var startups []*Startup
	for i, preview := range previews {
		if i >= 20 {
			break
		}

		startup, err := ScrapeCompany(ctx, preview.Slug)
		if err != nil {
			log.Printf("Error scraping %s: %v", preview.Slug, err)
			continue
		}

		if startup.Tagline == "" {
			startup.Tagline = preview.Tagline
		}

		startups = append(startups, startup)
	}
	SaveCache(startups)
	return startupsLoadedMsg{startups: startups}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, cmd

	case startupsLoadedMsg:
		m.startups = msg.startups
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		if m.filterMode {
			switch msg.String() {
			case "enter":
				m.filterMode = false
				m.filterInput = ""
			case "esc":
				m.filterMode = false
				m.filterInput = ""
			case "backspace":
				if len(m.filterInput) > 0 {
					m.filterInput = m.filterInput[:len(m.filterInput)-1]
				}
			default:
				m.filterInput += msg.String()
			}
			return m, cmd
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "j", "down":
			if m.selectedIndex < len(m.startups)-1 {
				m.selectedIndex++
			}
		case "k", "up":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		case "w":
			if len(m.startups) > 0 {
				go openURL(m.startups[m.selectedIndex].WebsiteURL)
			}
		case "c":
			if len(m.startups) > 0 {
				go openURL(m.startups[m.selectedIndex].JobsURL)
			}
		case ":":
			m.filterMode = true
			m.filterInput = ""
		}
	}

	return m, cmd
}

func (m model) View() string {
	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(bloombergOrange).
			Bold(true).
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center)
		text := fmt.Sprintf("%s LOADING MARKET DATA %s", m.spinner.View(), m.spinner.View())
		return loadingStyle.Render(text)
	}

	if len(m.startups) == 0 {
		return "No startups found"
	}
	header := renderHeader(m.width)
	headerHeight := lipgloss.Height(header)
	availableHeight := m.height - headerHeight - 3
	previewWidth := m.width / 4
	listWidth := m.width / 3
	metadataWidth := m.width - previewWidth - listWidth

	selectedStartup := m.startups[m.selectedIndex]

	preview := renderPreview(selectedStartup, previewWidth, availableHeight)

	list := renderList(m.startups, m.selectedIndex, listWidth, availableHeight)

	metadata := renderMetadata(selectedStartup, metadataWidth, availableHeight)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		preview,
		list,
		metadata,
	)

	var statusBar string
	if m.filterMode {
		statusBar = fmt.Sprintf(":%s_", m.filterInput)
	} else {
		statusBar = "j/k navigate • w website • c careers • : filter • q quit • powered by startups.gallery"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(bloombergBg).
		Background(bloombergOrange).
		Width(m.width).
		Bold(true).
		Padding(0, 2)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		statusStyle.Render(statusBar),
	)
}
func renderHeader(width int) string {
	now := time.Now().Format("2006-01-09 15:04:05")

	leftSection := lipgloss.NewStyle().
		Foreground(bloombergOrange).
		Bold(true).
		Render("STARTUPS.TERMINAL")

	rightSection := lipgloss.NewStyle().
		Foreground(bloombergGray).
		Render(fmt.Sprintf("[LIVE] %s", now))

	spacer := strings.Repeat(" ", width-lipgloss.Width(leftSection)-lipgloss.Width(rightSection))

	header := lipgloss.JoinHorizontal(lipgloss.Top, leftSection, spacer, rightSection)

	headerStyle := lipgloss.NewStyle().
		Background(bloombergBg).
		Foreground(bloombergText).
		Width(width).
		Padding(0, 2).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBottom(true).
		BorderForeground(bloombergOrange)

	return headerStyle.Render(header)
}

func renderPreview(startup *Startup, width, height int) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(bloombergOrange).
		Bold(true).
		Underline(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(bloombergGray)

	valueStyle := lipgloss.NewStyle().
		Foreground(bloombergText)

	content := fmt.Sprintf(
		"%s\n\n"+
			"%s\n\n"+
			"━━━━━━━━━━━━━━━━━━\n\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s",
		titleStyle.Render(startup.Name),
		lipgloss.NewStyle().Foreground(bloombergGreen).Render(startup.Tagline),
		labelStyle.Render("LOC:"), valueStyle.Render(startup.Location),
		labelStyle.Render("STG:"), valueStyle.Render(startup.FundingStage),
		labelStyle.Render("IND:"), valueStyle.Render(startup.Industry),
	)

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(bloombergBlue).
		Background(bloombergBg).
		Foreground(bloombergText).
		Padding(1, 2)

	return style.Render(content)
}

func renderList(startups []*Startup, selected int, width, height int) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(bloombergOrange).
		Bold(true).
		Padding(0, 0, 1, 0)

	title := titleStyle.Render(fmt.Sprintf("COMPANIES [%d]", len(startups)))

	var items []string
	items = append(items, title)
	items = append(items, lipgloss.NewStyle().Foreground(bloombergGray).Render(strings.Repeat("─", width-4)))

	startIdx := 0
	endIdx := len(startups)

	viewportHeight := height - 6
	if selected > viewportHeight/2 {
		startIdx = selected - viewportHeight/2
		endIdx = startIdx + viewportHeight
		if endIdx > len(startups) {
			endIdx = len(startups)
			startIdx = endIdx - viewportHeight
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	maxNameWidth := width - 25

	for i := startIdx; i < endIdx && i < len(startups); i++ {
		startup := startups[i]

		name := startup.Name
		if len(name) > maxNameWidth {
			name = name[:maxNameWidth-3] + "..."
		}

		stage := startup.FundingStage
		if len(stage) > 15 {
			stage = stage[:12] + "..."
		}

		if i == selected {
			itemText := fmt.Sprintf("▶ %-*s %s", maxNameWidth, name, stage)

			if len(itemText) > width-4 {
				itemText = itemText[:width-4]
			}

			items = append(items, lipgloss.NewStyle().
				Foreground(bloombergBg).
				Background(bloombergOrange).
				Bold(true).
				Width(width-4).
				Render(itemText))
		} else {
			itemText := fmt.Sprintf("  %-*s %s", maxNameWidth, name, stage)

			if len(itemText) > width-4 {
				itemText = itemText[:width-4]
			}

			items = append(items, lipgloss.NewStyle().
				Foreground(bloombergText).
				Render(itemText))
		}
	}

	if len(startups) > viewportHeight {
		scrollInfo := fmt.Sprintf("[%d-%d of %d]", startIdx+1, endIdx, len(startups))
		items = append(items, "")
		items = append(items, lipgloss.NewStyle().
			Foreground(bloombergGray).
			Render(scrollInfo))
	}

	content := strings.Join(items, "\n")

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(bloombergBlue).
		Background(bloombergBg).
		Padding(1, 2)

	return style.Render(content)
}
func renderMetadata(startup *Startup, width, height int) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(bloombergOrange).
		Bold(true)

	sectionStyle := lipgloss.NewStyle().
		Foreground(bloombergGreen).
		Bold(true).
		MarginTop(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(bloombergGray)

	valueStyle := lipgloss.NewStyle().
		Foreground(bloombergText)

	descWords := strings.Fields(startup.Description)
	var descLines []string
	currentLine := ""
	maxWidth := width - 6

	for _, word := range descWords {
		if len(currentLine)+len(word)+1 > maxWidth {
			descLines = append(descLines, currentLine)
			currentLine = word
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}
	if currentLine != "" {
		descLines = append(descLines, currentLine)
	}

	content := fmt.Sprintf(
		"%s\n"+
			"%s\n\n"+
			"%s\n"+
			"%s\n\n"+
			"%s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %s\n\n"+
			"%s\n"+
			"%s\n"+
			"%s",
		titleStyle.Render(startup.Name),
		lipgloss.NewStyle().Foreground(bloombergGreen).Italic(true).Render(startup.Tagline),
		sectionStyle.Render("DESCRIPTION"),
		strings.Join(descLines, "\n"),
		sectionStyle.Render("DETAILS"),
		labelStyle.Render("Location:"), valueStyle.Render(startup.Location),
		labelStyle.Render("Stage:   "), valueStyle.Render(startup.FundingStage),
		labelStyle.Render("Industry:"), valueStyle.Render(startup.Industry),
		labelStyle.Render("Work:    "), valueStyle.Render(startup.WorkType),
		labelStyle.Render("Size:    "), valueStyle.Render(startup.TeamSize),
		sectionStyle.Render("ACTIONS"),
		lipgloss.NewStyle().Foreground(bloombergYellow).Render("[W] Visit Website"),
		lipgloss.NewStyle().Foreground(bloombergYellow).Render("[C] View Careers"),
	)

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(bloombergBlue).
		Background(bloombergBg).
		Padding(1, 2)

	return style.Render(content)
}
