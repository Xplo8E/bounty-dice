// xplo8e 
package main

import (
	"bufio"
	"github.com/xplo8e/bounty-dice/pkg/client"
	"github.com/xplo8e/bounty-dice/pkg/random"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/sw33tLie/bbscope/pkg/scope"
)

// Mission struct to hold state
type Mission struct {
	Program     scope.ProgramData `json:"program"`
	StartDate   time.Time         `json:"start_date"`
	EndDate     time.Time         `json:"end_date"`
	RerollCount int               `json:"reroll_count"`
	Duration    int               `json:"duration"`
}

const (
	maxRerolls      = 5
	minMissionDays = 15
	maxMissionDays = 30
)

var (
	missionFilePath string

	// Styles
	styleAppTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#874BFD")).SetString("Bounty Dice")
	styleBox      = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("#874BFD")).Padding(1, 2).Width(80)
	styleHeader   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#874BFD")).Padding(0, 1)
	styleTarget   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8F8FF"))
	styleURL      = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Underline(true)
	styleLabel    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8A8A8A"))
	styleScope    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A9A9A9"))
	styleCommitment = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#FFD700"))
	styleProbability = lipgloss.NewStyle().Foreground(lipgloss.Color("#228B22"))
	styleWarning  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF6347"))

	// Motivational Messages
	motivations = []string{
		"For the next few weeks, this is your world. Explore every corner.",
		"A focused hunter finds the rarest beasts. Good luck.",
		"Commit to the target. The bugs will reveal themselves to the dedicated.",
		"A focused sprint can change everything. Let the hunt begin.",
		"Your mission, should you choose to accept it. This message will not self-destruct.",
	}
)

func init() {
	missionFilePath = letsStart()
}

func letsStart() string {
	en := "LmJvdW50eV9taXNzaW9uLmpzb24=" 
	de, err := base64.StdEncoding.DecodeString(en)
	fileName := string(de)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error finding home directory:", err)
		return fileName
	}
	return filepath.Join(homeDir, fileName)
}

func main() {
	reroll := flag.Bool("reroll", false, "Abandon your current mission and start a new one.")
	bounty := flag.Bool("bounty", false, "Roll the dice only for programs that offer bounties")
	scopeFilter := flag.String("scope", "all", "Filter by scope (e.g., url, cidr, mobile, android, apple, ai, other, hardware, code, executable)")
	duration := flag.Int("duration", 15, "Set the mission duration in days (min: 15, max: 30)")
	flag.Parse()

	if *duration < minMissionDays || *duration > maxMissionDays {
		fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			styleHeader.Render("INVALID DURATION"),
			"",
			styleWarning.Render(fmt.Sprintf("Mission duration must be between %d and %d days.", minMissionDays, maxMissionDays)),
			"A true focus sprint requires a meaningful, but achievable, timeframe.",
		)))
		return
	}

	mission, err := loadMission()

	// Check for existing mission
	if !*reroll && err == nil {
		if time.Now().Before(mission.EndDate) {
			displayActiveMission(mission)
			return
		} else {
			// Mission is complete, reset reroll counter
			mission.RerollCount = 0
			saveMission(mission)
		}
	}

	currentRerollCount := 0
	if err == nil {
		currentRerollCount = mission.RerollCount
	}

	if *reroll {
		if currentRerollCount >= maxRerolls {
			fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
				styleHeader.Render("REROLL LOCKOUT"),
				"",
				styleWarning.Render("You have exhausted all your reroll charges."),
				"Complete your current mission to reset your focus.",
			)))
			return
		}

		fmt.Println(styleWarning.Render("WARNING: A true hunter values focus. Rerolling is a sign of a wandering mind."))
		fmt.Printf("You are about to use reroll charge %d of %d.\n\n", currentRerollCount+1, maxRerolls)

		rerollPrompts := []string{
			"Focus is a hunter's greatest asset. Are you sure you want to dull your edge?",
			"The rarest bugs hide from wandering eyes. Stick to the path. Abandon mission?",
			"Discipline is built through commitment. Don't break your streak. Are you absolutely certain?",
			"FINAL WARNING. This action will consume a focus charge. There is no honor in retreat. Proceed?",
		}

		confirmed := true
		for i, prompt := range rerollPrompts {
			fmt.Printf("%s (%d/4) (y/n): ", styleWarning.Render(prompt), i+1)
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(input)) != "y" {
				confirmed = false
				break
			}
		}

		if !confirmed {
			fmt.Println("\nMission aborted. Your current mission remains active. A wise choice.")
			return
		}

		fmt.Println(styleCommitment.Render("\nFocus broken. Reroll charge consumed..."))
		os.Remove(missionFilePath)
		currentRerollCount++
	}

	// --- Start New Mission ---
	apiUser := os.Getenv("HACKERONE_API_USER")
	apiToken := os.Getenv("HACKERONE_API_TOKEN")

	if apiUser == "" || apiToken == "" {
		fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			styleAppTitle.String(),
			styleLabel.Render("\nError: HACKERONE_API_USER and HACKERONE_API_TOKEN must be set."),
		)))
		os.Exit(1)
	}

	authString := apiUser + ":" + apiToken
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(authString))

	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Rolling the dice..."
	s.Color("fgHiCyan")
	s.Start()

	programs, err := client.GetPrograms(encodedAuth, *bounty, *scopeFilter)
	if err != nil {
		s.Stop()
		fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			styleAppTitle.String(),
			styleLabel.Render(fmt.Sprintf("\nError fetching programs: %v", err)),
		)))
		os.Exit(1)
	}
	s.Stop()

	if len(programs) == 0 {
		fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			styleAppTitle.String(),
			styleLabel.Render("\nNo programs found matching your criteria. Try different filters!"),
		)))
		os.Exit(0)
	}

	randomProgram, err := random.Select(programs)
	if err != nil {
		fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			styleAppTitle.String(),
			styleLabel.Render(fmt.Sprintf("\nError selecting random program: %v", err)),
		)))
		os.Exit(1)
	}

	// Create and save the new mission
	newMission := Mission{
		Program:     randomProgram,
		StartDate:   time.Now(),
		EndDate:     time.Now().Add(time.Duration(*duration) * 24 * time.Hour),
		RerollCount: currentRerollCount,
		Duration:    *duration,
	}
	saveMission(newMission)

	// Display the new mission briefing
	displayNewMission(newMission, len(programs))
}

func loadMission() (Mission, error) {
	var mission Mission
	data, err := os.ReadFile(missionFilePath)
	if err != nil {
		return Mission{}, err
	}
	err = json.Unmarshal(data, &mission)
	return mission, err
}

func saveMission(mission Mission) {
	data, err := json.MarshalIndent(mission, "", "  ")
	if err != nil {
		fmt.Println("Error saving mission:", err)
		return
	}
	os.WriteFile(missionFilePath, data, 0644)
}

func displayActiveMission(mission Mission) {
	daysRemaining := int(math.Ceil(time.Until(mission.EndDate).Hours() / 24))

	// For backward compatibility with missions created before the duration field
	missionDuration := mission.Duration
	if missionDuration == 0 {
		missionDuration = int(mission.EndDate.Sub(mission.StartDate).Hours() / 24)
	}

	header := styleHeader.Render("ACTIVE MISSION BRIEFING")

	missionDetails := lipgloss.JoinVertical(lipgloss.Left,
		styleLabel.Render("CURRENT TARGET:"),
		styleTarget.Render(mission.Program.Url),
		styleURL.Render(mission.Program.Url),
		"",
		styleLabel.Render("MISSION DURATION:"),
		fmt.Sprintf("%d Days", missionDuration),
		"",
		styleLabel.Render("DAYS REMAINING:"),
		fmt.Sprintf("%d days", daysRemaining),
	)

	commitment := styleCommitment.Render("Your focus is sharp. Your dedication is unwavering. Keep hunting.")

	output := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		missionDetails,
		"",
		commitment,
		"",
		styleLabel.Render(fmt.Sprintf("Reroll charges remaining: %d", maxRerolls-mission.RerollCount)),
		styleLabel.Render("To start a new mission, use the -reroll flag."),
	)

	fmt.Println(styleBox.Render(output))
}

func displayNewMission(mission Mission, totalPrograms int) {
	rand.Seed(time.Now().UnixNano())
	motivation := motivations[rand.Intn(len(motivations))]
	probability := (1.0 / float64(totalPrograms)) * 100

	var scopeLines []string
	for i, s := range mission.Program.InScope {
		if i >= 5 {
			scopeLines = append(scopeLines, styleScope.Render("  ...and more"))
			break
		}
		scopeLines = append(scopeLines, styleScope.Render("  • "+s.Target))
	}

	header := styleHeader.Render("YOUR NEXT MISSION")
	missionDetails := lipgloss.JoinVertical(lipgloss.Left,
		styleLabel.Render("TARGET:"),
		styleTarget.Render(mission.Program.Url),
		styleURL.Render(mission.Program.Url),
		"",
		styleLabel.Render("MISSION DURATION:"),
		fmt.Sprintf("%d Days", mission.Duration),
		"",
		styleLabel.Render("MATCHING SCOPE:"),
		strings.Join(scopeLines, "\n"),
	)

	commitment := styleCommitment.Render("Pledge to focus solely on this target. Let your curiosity guide you.")
	probMsg := styleProbability.Render(fmt.Sprintf("Encounter Rate: %.2f%%. This target was chosen from %d possibilities.", probability, totalPrograms))

	output := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		missionDetails,
		"",
		commitment,
		"",
		probMsg,
		"",
		styleLabel.Render(motivation),
	)

	fmt.Println(styleBox.Render(output))
}
