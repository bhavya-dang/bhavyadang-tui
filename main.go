package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
)

// ─── Data Types ───────────────────────────────────────────────────────────────

type Portfolio struct {
	Header struct {
		ASCII        string `json:"ascii"`
		Tagline      string `json:"tagline"`
		Availability string `json:"availability"`
	} `json:"header"`
	Home struct {
		Greeting string   `json:"greeting"`
		Intro    []string `json:"intro"`
		Stats    []Stat   `json:"stats"`
		Links    []Link   `json:"links"`
		Shell    struct {
			Command string `json:"command"`
			Hint    string `json:"hint"`
		} `json:"shell"`
	} `json:"home"`
	About struct {
		Bio       []string `json:"bio"`
		Skills    Skills   `json:"skills"`
		Interests []string `json:"interests"`
	} `json:"about"`
	Projects []Project `json:"projects"`
	Blog     Blog      `json:"blog"`
	Contact  Contact   `json:"contact"`
}

type Stat struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Color string `json:"color"`
}

type Link struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Color string `json:"color"`
}

type Skills struct {
	Languages []string `json:"languages"`
	Frontend  []string `json:"frontend"`
	Backend   []string `json:"backend"`
	Tools     []string `json:"tools"`
}

type Project struct {
	Name      string   `json:"name"`
	Desc      string   `json:"desc"`
	Tags      []string `json:"tags"`
	Link      string   `json:"link"`
	DemoURL   string   `json:"demo_url"`
	GitHubURL string   `json:"github_url"`
	Status    string   `json:"status"`
	Order     int      `json:"order"`
	Star      bool     `json:"star"`
	Color     string   `json:"color"`
}

type Blog struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

type Contact struct {
	Header  string          `json:"header"`
	Pitch   []string        `json:"pitch"`
	Methods []ContactMethod `json:"methods"`
	Share   string          `json:"share"`
}

type ContactMethod struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Color string `json:"color"`
}

// ─── Networking ──────────────────────────────────────────────────────────────

const (
	host = "0.0.0.0"
	port = "23234"
)

var portfolio Portfolio

func loadPortfolio() {
	b, err := os.ReadFile("portfolio.json")
	if err != nil {
		log.Fatalf("failed to read portfolio.json: %v", err)
	}
	if err := json.Unmarshal(b, &portfolio); err != nil {
		log.Fatalf("failed to parse portfolio.json: %v", err)
	}
	if err := loadProjectsFromCSV("projectsData.csv"); err != nil {
		log.Printf("could not load projectsData.csv, keeping JSON projects: %v", err)
	}
}

func loadProjectsFromCSV(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true

	rows, err := r.ReadAll()
	if err != nil {
		return err
	}
	if len(rows) <= 1 {
		return nil
	}

	palette := []string{"accent", "cyan", "green", "purple", "orange", "yellow"}
	var projects []Project
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 8 {
			continue
		}
		order, _ := strconv.Atoi(strings.TrimSpace(row[5]))
		tags := splitTags(row[7])
		colorName := palette[len(projects)%len(palette)]
		name := strings.TrimSpace(row[0])
		demo := strings.TrimSpace(row[1])
		desc := strings.TrimSpace(row[2])
		gh := strings.TrimSpace(row[3])
		projects = append(projects, Project{
			Name:      name,
			Desc:      desc,
			Tags:      tags,
			Link:      demo,
			DemoURL:   demo,
			GitHubURL: gh,
			Status:    "",
			Order:     order,
			Color:     colorName,
		})
	}

	sort.SliceStable(projects, func(i, j int) bool {
		if projects[i].Order == projects[j].Order {
			return projects[i].Name < projects[j].Name
		}
		return projects[i].Order < projects[j].Order
	})

	portfolio.Projects = projects
	return nil
}

func main() {
	loadPortfolio()
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%s", host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
			activeterm.Middleware(),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalf("Could not start server: %s", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Starting SSH server on %s:%s", host, port)

	go func() {
		if err = s.ListenAndServe(); err != nil {
			log.Fatalf("Could not start server: %s", err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatalf("Could not stop server gracefully: %s", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	m := NewModel(pty.Window.Width, pty.Window.Height)
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// ─── Styles & Colors ─────────────────────────────────────────────────────────

var (
	colorBg     = lipgloss.Color("#0d0f14")
	colorAccent = lipgloss.Color("#7aa2f7")
	colorGreen  = lipgloss.Color("#9ece6a")
	colorPurple = lipgloss.Color("#bb9af7")
	colorOrange = lipgloss.Color("#ff9e64")
	colorRed    = lipgloss.Color("#f7768e")
	colorCyan   = lipgloss.Color("#7dcfff")
	colorMuted  = lipgloss.Color("#565f89")
	colorFg     = lipgloss.Color("#c0caf5")
	colorYellow = lipgloss.Color("#e0af68")
	colorBorder = lipgloss.Color("#1a1b26")
	colorHL     = lipgloss.Color("#1e2030")

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			PaddingLeft(1)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBg).
			Background(colorAccent).
			Padding(0, 2)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 2)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	styleLabel = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(12)

	styleValue = lipgloss.NewStyle().
			Foreground(colorFg)

	styleBadge = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorGreen).
			Padding(0, 1).
			Bold(true)

	styleLink = lipgloss.NewStyle().
			Foreground(colorCyan).
			Underline(true)

	styleSectionHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPurple)

	styleProjectCard = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorHL).
				Padding(1, 2).
				MarginBottom(1)

	styleHighlight = lipgloss.NewStyle().
			Foreground(colorYellow)

	styleSubtle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	styleStatusBar = lipgloss.NewStyle().
			Background(colorHL).
			Foreground(colorMuted).
			Padding(0, 2)

	styleKey = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)
)

func colorFromName(name string) lipgloss.Color {
	switch strings.ToLower(name) {
	case "accent":
		return colorAccent
	case "green":
		return colorGreen
	case "purple":
		return colorPurple
	case "orange":
		return colorOrange
	case "red":
		return colorRed
	case "cyan":
		return colorCyan
	case "muted":
		return colorMuted
	case "fg":
		return colorFg
	case "yellow":
		return colorYellow
	case "border":
		return colorBorder
	case "hl":
		return colorHL
	case "bg":
		return colorBg
	default:
		return colorAccent
	}
}

// ─── Model ────────────────────────────────────────────────────────────────────

type tab int

const (
	tabHome tab = iota
	tabAbout
	tabProjects
	tabBlog
	tabContact
)

var tabNames = []string{"  home  ", "  about  ", "  projects  ", "  blog  ", "  contact  "}

var loadingFrames = []string{
	"    ╭────╮",
	"    │ ⌛ │",
	"    ╰────╯",
	"    ╭────╮",
	"    │ ⏳ │",
	"    ╰────╯",
}

type Model struct {
	activeTab    tab
	width        int
	height       int
	scroll       int
	loadingFrame int
}

func NewModel(w, h int) Model {
	return Model{
		activeTab: tabHome,
		width:     w,
		height:    h,
	}
}

func (m Model) Init() tea.Cmd {
	return loadingTick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case time.Time:
		if m.width == 0 {
			m.loadingFrame = (m.loadingFrame + 1) % len(loadingFrames)
			return m, loadingTick()
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "l", "right":
			if int(m.activeTab) < len(tabNames)-1 {
				m.activeTab++
				m.scroll = 0
			}
		case "shift+tab", "h", "left":
			if m.activeTab > 0 {
				m.activeTab--
				m.scroll = 0
			}
		case "1":
			m.activeTab = tabHome
			m.scroll = 0
		case "2":
			m.activeTab = tabAbout
			m.scroll = 0
		case "3":
			m.activeTab = tabProjects
			m.scroll = 0
		case "4":
			m.activeTab = tabBlog
			m.scroll = 0
		case "5":
			m.activeTab = tabContact
			m.scroll = 0
		case "j", "down":
			m.scroll++
		case "k", "up":
			if m.scroll > 0 {
				m.scroll--
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		frame := loadingFrames[m.loadingFrame%len(loadingFrames)]
		return styleSubtle.Copy().Margin(1, 2).Render(frame)
	}

	header := m.renderHeader()
	tabs := m.renderTabs()
	content := m.renderContent()
	statusBar := m.renderStatusBar()

	headerH := lipgloss.Height(header)
	tabsH := lipgloss.Height(tabs)
	statusH := lipgloss.Height(statusBar)
	contentH := m.height - headerH - tabsH - statusH - 1

	contentLines := splitLines(content)
	if m.scroll > len(contentLines)-contentH {
		m.scroll = max(0, len(contentLines)-contentH)
	}
	visibleLines := contentLines
	if m.scroll < len(contentLines) {
		visibleLines = contentLines[m.scroll:]
	}
	if len(visibleLines) > contentH {
		visibleLines = visibleLines[:contentH]
	}
	scrolledContent := joinLines(visibleLines)
	scrolledContent = lipgloss.NewStyle().Height(contentH).Render(scrolledContent)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabs,
		scrolledContent,
		statusBar,
	)
}

// ─── Rendering ────────────────────────────────────────────────────────────────

func (m Model) renderHeader() string {
	asciiStyled := lipgloss.NewStyle().
		Foreground(colorAccent).
		Render(portfolio.Header.ASCII)

	tagline := lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true).
		PaddingLeft(2).
		Render(portfolio.Header.Tagline)

	right := lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true).
		Align(lipgloss.Right).
		PaddingRight(2).
		Render(portfolio.Header.Availability)

	leftW := m.width - lipgloss.Width(right) - 2
	leftBlock := lipgloss.NewStyle().Width(leftW).Render(
		lipgloss.JoinVertical(lipgloss.Left, asciiStyled, tagline),
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, right)

	divider := lipgloss.NewStyle().
		Foreground(colorBorder).
		Render(repeatStr("─", m.width))

	return lipgloss.JoinVertical(lipgloss.Left, topRow, divider)
}

func (m Model) renderTabs() string {
	tabs := ""
	for i, name := range tabNames {
		var t string
		num := styleSubtle.Render(fmt.Sprintf("%d", i+1))
		if tab(i) == m.activeTab {
			t = styleTabActive.Render(fmt.Sprintf("%s %s", num, name))
		} else {
			t = styleTabInactive.Render(fmt.Sprintf("%s %s", num, name))
		}
		tabs += t
	}

	bar := lipgloss.NewStyle().
		Background(colorBorder).
		Width(m.width).
		Render(tabs)

	return bar
}

func (m Model) renderStatusBar() string {
	left := styleKey.Render("←→ / h l") + styleStatusBar.Render(" navigate tabs  ") +
		styleKey.Render("j k") + styleStatusBar.Render(" scroll  ") +
		styleKey.Render("1-5") + styleStatusBar.Render(" jump  ") +
		styleKey.Render("q") + styleStatusBar.Render(" quit")

	right := styleStatusBar.Render("bhavyadang.in")

	rightW := lipgloss.Width(right)
	leftBlock := lipgloss.NewStyle().Width(m.width - rightW).Render(left)
	return lipgloss.NewStyle().Background(colorHL).Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, right))
}

func (m Model) renderContent() string {
	inner := m.width - 4
	switch m.activeTab {
	case tabHome:
		return m.renderHome(inner)
	case tabAbout:
		return m.renderAbout(inner)
	case tabProjects:
		return m.renderProjects(inner)
	case tabBlog:
		return m.renderBlog()
	case tabContact:
		return m.renderContact(inner)
	}
	return ""
}

// ─── Home ─────────────────────────────────────────────────────────────────────

func (m Model) renderHome(w int) string {
	greeting := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorFg).
		Render(portfolio.Home.Greeting)

	intro := lipgloss.NewStyle().
		Foreground(colorFg).
		Width(w).
		Render(joinParagraph(portfolio.Home.Intro))

	statCards := []string{}
	for _, st := range portfolio.Home.Stats {
		statCards = append(statCards, m.statCard(st.Label, st.Value, colorFromName(st.Color)))
	}
	stats := lipgloss.JoinHorizontal(lipgloss.Top, statCards...)

	featuredLines := []string{styleSectionHeader.Render("─── Quick Links")}
	for _, l := range portfolio.Home.Links {
		arrow := lipgloss.NewStyle().Foreground(colorAccent).Render("  →")
		featuredLines = append(featuredLines, fmt.Sprintf("%s  %s", arrow,
			styleValue.Render(fmt.Sprintf("%-14s", l.Label))+styleLink.Render(l.Value)))
	}
	featured := strings.Join(featuredLines, "\n")

	shell := lipgloss.NewStyle().
		Foreground(colorMuted).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorHL).
		Padding(0, 2).
		Render(
			styleSubtle.Render("$ ") +
				lipgloss.NewStyle().Foreground(colorGreen).Render(portfolio.Home.Shell.Command) + " " +
				lipgloss.NewStyle().Foreground(colorFg).Render(portfolio.Home.Shell.Hint),
		)

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(greeting),
		lipgloss.NewStyle().PaddingLeft(2).Render(intro),
		lipgloss.NewStyle().PaddingLeft(2).Render(stats),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(featured),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(shell),
	)
}

func (m Model) statCard(label, value string, c lipgloss.Color) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorHL).
		Padding(0, 2).
		MarginRight(2).
		Render(
			lipgloss.NewStyle().Foreground(colorMuted).Render(label) + "\n" +
				lipgloss.NewStyle().Foreground(c).Bold(true).Render(value),
		)
}

// ─── About ────────────────────────────────────────────────────────────────────

type skillItem struct {
	name  string
	color lipgloss.Color
}

func (m Model) renderAbout(w int) string {
	bio := styleBox.Width(w).Render(
		styleSectionHeader.Render("About Me") + "\n\n" +
			lipgloss.NewStyle().Foreground(colorFg).Render(joinParagraph(portfolio.About.Bio)),
	)

	skills := styleBox.Width(w).Render(
		styleSectionHeader.Render("Skills & Tech") + "\n\n" +
			m.skillRow("Languages", toSkillItems(portfolio.About.Skills.Languages)) + "\n" +
			m.skillRow("Frontend", toSkillItems(portfolio.About.Skills.Frontend)) + "\n" +
			m.skillRow("Backend", toSkillItems(portfolio.About.Skills.Backend)) + "\n" +
			m.skillRow("Tools", toSkillItems(portfolio.About.Skills.Tools)),
	)

	interestBadges := []string{}
	for _, in := range portfolio.About.Interests {
		interestBadges = append(interestBadges, m.interestBadge(in, colorAccent))
	}
	interests := styleBox.Width(w).Render(
		styleSectionHeader.Render("Interests") + "\n\n" +
			lipgloss.JoinHorizontal(lipgloss.Top, interestBadges...),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(bio),
		lipgloss.NewStyle().PaddingLeft(2).Render(skills),
		lipgloss.NewStyle().PaddingLeft(2).Render(interests),
	)
}

func toSkillItems(names []string) []skillItem {
	items := make([]skillItem, 0, len(names))
	palette := []lipgloss.Color{colorCyan, colorAccent, colorGreen, colorPurple, colorOrange, colorYellow, colorFg}
	for i, n := range names {
		items = append(items, skillItem{name: n, color: palette[i%len(palette)]})
	}
	return items
}

func (m Model) skillRow(label string, items []skillItem) string {
	row := styleLabel.Render(label + ":")
	for _, s := range items {
		row += lipgloss.NewStyle().
			Foreground(s.color).
			Background(colorHL).
			Padding(0, 1).
			MarginLeft(1).
			Render(s.name)
	}
	return row
}

func (m Model) interestBadge(name string, c lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(c).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c).
		Padding(0, 1).
		Render(name)
}

// ─── Projects ─────────────────────────────────────────────────────────────────

func (m Model) renderProjects(w int) string {
	header := styleSectionHeader.Render("Projects") + "  " +
		styleSubtle.Render(fmt.Sprintf("(%d total)", len(portfolio.Projects)))

	cards := ""
	for _, p := range portfolio.Projects {
		cards += m.projectCard(p, w) + "\n"
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(cards),
	)
}

func (m Model) projectCard(p Project, w int) string {
	badge := ""
	if p.Star {
		badge = " " + lipgloss.NewStyle().Foreground(colorYellow).Render("★")
	}

	pColor := colorFromName(p.Color)
	statusBadge := ""
	if p.Status != "" {
		statusBadge = " " + styleBadge.Copy().Background(pColor).Foreground(colorBg).Render(p.Status)
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(pColor).Render(p.Name) + badge + statusBadge
	desc := lipgloss.NewStyle().Foreground(colorFg).Width(w - 8).Render(p.Desc)

	tags := ""
	for _, t := range p.Tags {
		tags += lipgloss.NewStyle().
			Foreground(colorBg).
			Background(pColor).
			Padding(0, 1).
			MarginRight(1).
			Render(t)
	}

	linkLines := []string{}
	if p.DemoURL != "" {
		linkLines = append(linkLines, styleLink.Render("Live: "+p.DemoURL))
	}
	if p.GitHubURL != "" {
		linkLines = append(linkLines, styleLink.Render("GitHub: "+p.GitHubURL))
	}
	if len(linkLines) == 0 && p.Link != "" {
		linkLines = append(linkLines, styleLink.Render("→ "+p.Link))
	}
	links := strings.Join(linkLines, "\n")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		desc,
		"",
		tags,
		"",
		links,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pColor).
		Padding(1, 2).
		Width(w).
		Render(content)
}

// ─── Blog ─────────────────────────────────────────────────────────────────────

func (m Model) renderBlog() string {
	header := styleSectionHeader.Render(portfolio.Blog.Title) + "\n" +
		styleSubtle.Render("  "+portfolio.Blog.Subtitle)

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
	)
}

// ─── Contact ──────────────────────────────────────────────────────────────────

func (m Model) renderContact(w int) string {
	header := styleSectionHeader.Render(portfolio.Contact.Header)

	mainCard := styleBox.Width(w).Render(
		lipgloss.NewStyle().Foreground(colorFg).Render(
			joinParagraph(portfolio.Contact.Pitch),
		) + "\n\n" +
			m.renderContactMethods(portfolio.Contact.Methods),
	)

	// Add this back when its deployed on ssh.bhavyadang.in

	// sshCard := styleBox.Width(w).Render(
	// 	styleSectionHeader.Render("Share this TUI") + "\n\n" +
	// 		lipgloss.NewStyle().Foreground(colorFg).Render("Know someone who'd enjoy this? Have them run:\n\n") +
	// 		lipgloss.NewStyle().
	// 			Background(colorHL).
	// 			Foreground(colorGreen).
	// 			Padding(0, 2).
	// 			Render(portfolio.Contact.Share),
	// )

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(header),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(mainCard),
		// lipgloss.NewStyle().PaddingLeft(2).Render(sshCard),
	)
}

func (m Model) renderContactMethods(methods []ContactMethod) string {
	lines := []string{}
	for _, cm := range methods {
		lines = append(lines, m.contactRow(cm.Label, cm.Value, colorFromName(cm.Color)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) contactRow(label, value string, c lipgloss.Color) string {
	l := styleLabel.Render(label + ":")
	v := lipgloss.NewStyle().Foreground(c).Render(value)
	return l + v
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}

func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func loadingTick() tea.Cmd {
	return tea.Tick(time.Second/6, func(t time.Time) tea.Msg {
		return t
	})
}

func splitTags(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func joinParagraph(lines []string) string {
	return strings.Join(lines, "\n")
}
