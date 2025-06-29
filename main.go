// xplo8e 
package main

import (
	"bufio"
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
	"github.com/xplo8e/bounty-dice/pkg/client"
	"github.com/xplo8e/bounty-dice/pkg/hq"
	"github.com/xplo8e/bounty-dice/pkg/random"
)

// Mission struct to hold state
type Mission struct {
	Program     scope.ProgramData `json:"program"`
	StartDate   time.Time         `json:"start_date"`
	EndDate     time.Time         `json:"end_date"`
	RerollCount int               `json:"reroll_count"`
	Duration    int               `json:"duration"`
	HQFindings  []string          `json:"hq_findings,omitempty"`
}

const (
	maxRerolls     = 5
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
	styleVerbose = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	// Motivational Messages
	motivations = []string{
		"For the next few weeks, this is your world. Explore every corner.",
		"A focused hunter finds the rarest beasts. Good luck.",
		"Commit to the target. The bugs will reveal themselves to the dedicated.",
		"A focused sprint can change everything. Let the hunt begin.",
		"Your mission, should you choose to accept it. This message will not self-destruct.",
	}
)

var verboseLog *bool

func logVerbose(format string, a ...interface{}) {
	if verboseLog != nil && *verboseLog {
		fmt.Println(styleVerbose.Render(fmt.Sprintf(format, a...)))
	}
}

func init() {
	missionFilePath = letsStart()
}

func letsStart() string {
	en := "LmJvdW50eV9taXNzaW9uLmpzb24="
	de, _ := base64.StdEncoding.DecodeString(en)
	fileName := string(de)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logVerbose("Error finding home directory: %v. Using current directory.", err)
		return fileName
	}
	path := filepath.Join(homeDir, fileName)
	logVerbose("Mission file path set to: %s", path)
	return path
}

func main() {
	reroll := flag.Bool("reroll", false, "Abandon your current mission and start a new one.")
	bounty := flag.Bool("bounty", false, "Roll the dice only for programs that offer bounties")
	scopeFilter := flag.String("scope", "all", "Filter by scope (e.g., url, cidr, mobile, android, apple, ai, other, hardware, code, executable)")
	duration := flag.Int("duration", 15, "Set the mission duration in days (min: 15, max: 30)")
	hqFlag := flag.Bool("hq", false, "Use high-quality program fetching logic")
	force := flag.Bool("force", false, "Force re-fetch list of programs and program data")
	minReq := flag.Int("min-req", 1, "Minimum number of identified features required to highlight a program in the output")
	sessionCookie := flag.String("session", "", "__Host-session cookie value from hackerone.com")
	csrfToken := flag.String("token", "", "X-Csrf-Token value")
	verbose := flag.Bool("v", false, "Enable verbose output")
	flag.Parse()

	verboseLog = verbose
	client.SetVerbose(*verbose)
	hq.SetVerbose(*verbose)
	random.SetVerbose(*verbose)
	logVerbose("Verbose mode enabled.")
	logVerbose("Reroll: %t, Bounty: %t, Scope: %s, Duration: %d, HQ: %t, Force: %t, Min-Req: %d", *reroll, *bounty, *scopeFilter, *duration, *hqFlag, *force, *minReq)
	if *sessionCookie != "" {
		logVerbose("HackerOne session cookie provided.")
	}
	if *csrfToken != "" {
		logVerbose("HackerOne CSRF token provided.")
	}

	var currentRerollCount int
	mission, err := loadMission()
	if err == nil {
		currentRerollCount = mission.RerollCount
		logVerbose("Loaded existing mission. Current reroll count: %d", currentRerollCount)
	} else {
		logVerbose("No existing mission found or error loading: %v", err)
	}

	// Check for existing mission FIRST, unless rerolling
	if !*reroll && err == nil {
		if time.Now().Before(mission.EndDate) {
			logVerbose("Active mission found and it's not expired. Displaying it.")
			displayActiveMission(mission)
			return // <-- Exit early
		} else {
			logVerbose("Mission found, but it has expired. Resetting reroll counter.")
			mission.RerollCount = 0
			saveMission(mission)
			currentRerollCount = 0 // Reset for the new roll
		}
	}

	if *hqFlag {
		logVerbose("High-quality mode activated.")
		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		s.Suffix = " Fetching programs...\n"
		s.Color("fgHiCyan")
		s.Start()

		apiUser := os.Getenv("HACKERONE_API_USER")
		apiToken := os.Getenv("HACKERONE_API_TOKEN")
		authString := apiUser + ":" + apiToken
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(authString))
		allPrograms, err := getPrograms(encodedAuth, *bounty, *scopeFilter)
		if err != nil {
			s.Stop()
			fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
				styleAppTitle.String(),
				styleLabel.Render(fmt.Sprintf("\nError fetching programs: %v", err)),
			)))
			os.Exit(1)
		}
		s.Stop()
		logVerbose("Fetched %d programs initially.", len(allPrograms))

		if len(allPrograms) == 0 {
			fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
				styleAppTitle.String(),
				styleLabel.Render("\nNo programs found matching your criteria. Try different filters!"),
			)))
			os.Exit(0)
		}

		var handles []string
		for _, p := range allPrograms {
			handles = append(handles, strings.TrimPrefix(p.Url, "https://hackerone.com/"))
		}
		logVerbose("Extracted %d handles for high-quality checking.", len(handles))

		s.Suffix = " Finding high-quality programs...\n"
        s.Start()
        session := hq.NewSession(*sessionCookie, *csrfToken)
        hqProgramsResult, err := hq.FetchAndCheck(session, handles, *force, *minReq)
        if err != nil {
            s.Stop()
            fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
                styleAppTitle.String(),
                styleLabel.Render(fmt.Sprintf("\nError fetching high-quality programs: %v", err)),
            )))
            os.Exit(1)
        }
        s.Stop()
        logVerbose("Found %d high-quality programs.", len(hqProgramsResult))

        var hqPrograms []scope.ProgramData
        var hqProgramMap = make(map[string][]string)

        for _, hqProg := range hqProgramsResult {
            hqProgramMap[hqProg.Handle] = hqProg.Findings
        }

        for _, p := range allPrograms {
            if _, ok := hqProgramMap[strings.TrimPrefix(p.Url, "https://hackerone.com/")]; ok {
                hqPrograms = append(hqPrograms, p)
            }
        }

        logVerbose("Filtered down to %d high-quality programs matching initial criteria.", len(hqPrograms))

        if len(hqPrograms) == 0 {
            fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
                styleAppTitle.String(),
                styleLabel.Render("\nNo high-quality programs found matching your criteria. Try different filters!"),
            )))
            os.Exit(0)
        }

        logVerbose("Selecting a random program from the high-quality list.")
        randomProgram, err := random.Select(hqPrograms)
        if err != nil {
            fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
                styleAppTitle.String(),
                styleLabel.Render(fmt.Sprintf("\nError selecting random program: %v", err)),
            )))
            os.Exit(1)
        }

        randomProgramHandle := strings.TrimPrefix(randomProgram.Url, "https://hackerone.com/")
        findings := hqProgramMap[randomProgramHandle]

        newMission := Mission{
            Program:     randomProgram,
            StartDate:   time.Now(),
            EndDate:     time.Now().Add(time.Duration(*duration) * 24 * time.Hour),
            RerollCount: currentRerollCount,
            Duration:    *duration,
            HQFindings:  findings,
        }
        saveMission(newMission)
        logVerbose("New mission created and saved for program: %s", randomProgram.Url)
        displayNewMission(newMission, len(hqPrograms))
        return
    }

	if *duration < minMissionDays || *duration > maxMissionDays {
		logVerbose("Duration %d is outside the allowed range (%d-%d days).", *duration, minMissionDays, maxMissionDays)
		fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			styleHeader.Render("INVALID DURATION"),
			"",
			styleWarning.Render(fmt.Sprintf("Mission duration must be between %d and %d days.", minMissionDays, maxMissionDays)),
			"A true focus sprint requires a meaningful, but achievable, timeframe.",
		)))
		return
	}

	if *reroll {
		logVerbose("Reroll flag is set. Initiating reroll process.")
		if currentRerollCount >= maxRerolls {
			logVerbose("User has reached the maximum reroll limit of %d.", maxRerolls)
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
			logVerbose("User aborted the reroll process.")
			fmt.Println("\nMission aborted. Your current mission remains active. A wise choice.")
			return
		}

		logVerbose("User confirmed reroll. Removing old mission file and incrementing reroll count.")
		fmt.Println(styleCommitment.Render("\nFocus broken. Reroll charge consumed..."))
		os.Remove(missionFilePath)
		currentRerollCount++
	}

	// --- Start New Mission ---
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " Rolling the dice...\n"
	s.Color("fgHiCyan")
	s.Start()

	apiUser := os.Getenv("HACKERONE_API_USER")
	apiToken := os.Getenv("HACKERONE_API_TOKEN")
	authString := apiUser + ":" + apiToken
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(authString))

	logVerbose("Fetching programs for a new mission.")
	programs, err := getPrograms(encodedAuth, *bounty, *scopeFilter)
	if err != nil {
		s.Stop()
		fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			styleAppTitle.String(),
			styleLabel.Render(fmt.Sprintf("\nError fetching programs: %v", err)),
		)))
		os.Exit(1)
	}
	s.Stop()
	logVerbose("Fetched %d programs.", len(programs))

	if len(programs) == 0 {
		fmt.Println(styleBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			styleAppTitle.String(),
			styleLabel.Render("\nNo programs found matching your criteria. Try different filters!"),
		)))
		os.Exit(0)
	}

	logVerbose("Selecting a random program.")
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
	logVerbose("New mission created and saved for program: %s", randomProgram.Url)

	// Display the new mission briefing
	displayNewMission(newMission, len(programs))
}

func getPrograms(encodedAuth string, bounty bool, scopeFilter string) ([]scope.ProgramData, error) {
	logVerbose("Getting programs with bounty: %t, scope: %s", bounty, scopeFilter)
	if os.Getenv("HACKERONE_API_USER") == "" || os.Getenv("HACKERONE_API_TOKEN") == "" {
		return nil, fmt.Errorf("HACKERONE_API_USER and HACKERONE_API_TOKEN must be set")
	}
	programs, err := client.GetPrograms(encodedAuth, bounty, scopeFilter)
	if err != nil {
		logVerbose("Error from client.GetPrograms: %v", err)
	} else {
		logVerbose("client.GetPrograms returned %d programs", len(programs))
	}
	return programs, err
}

func loadMission() (Mission, error) {
	logVerbose("Attempting to load mission from %s", missionFilePath)
	var mission Mission
	data, err := os.ReadFile(missionFilePath)
	if err != nil {
		logVerbose("Failed to read mission file: %v", err)
		return Mission{}, err
	}
	err = json.Unmarshal(data, &mission)
	if err != nil {
		logVerbose("Failed to unmarshal mission JSON: %v", err)
	} else {
		logVerbose("Successfully loaded mission for program: %s", mission.Program.Url)
	}
	return mission, err
}

func saveMission(mission Mission) {
	logVerbose("Saving mission for program: %s to %s", mission.Program.Url, missionFilePath)
	data, err := json.MarshalIndent(mission, "", "  ")
	if err != nil {
		logVerbose("Error marshalling mission to JSON: %v", err)
		fmt.Println("Error saving mission:", err)
		return
	}
	err = os.WriteFile(missionFilePath, data, 0644)
	if err != nil {
		logVerbose("Error writing mission file: %v", err)
		fmt.Println("Error saving mission:", err)
	} else {
		logVerbose("Mission saved successfully.")
	}
}

func displayActiveMission(mission Mission) {
	logVerbose("Displaying active mission for program: %s", mission.Program.Url)
	daysRemaining := int(math.Ceil(time.Until(mission.EndDate).Hours() / 24))

	// For backward compatibility with missions created before the duration field
	missionDuration := mission.Duration
	if missionDuration == 0 {
		missionDuration = int(mission.EndDate.Sub(mission.StartDate).Hours() / 24)
		logVerbose("Mission duration not found, calculating from dates: %d days", missionDuration)
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
	logVerbose("Displaying new mission for program: %s", mission.Program.Url)
	logVerbose("Total programs considered for this roll: %d", totalPrograms)
	motivation := motivations[rand.Intn(len(motivations))]
	probability := (1.0 / float64(totalPrograms)) * 100
	logVerbose("Calculated probability: %.2f%%", probability)

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
