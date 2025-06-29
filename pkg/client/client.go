package client

import (
	"fmt"
	"github.com/sw33tLie/bbscope/pkg/platforms/hackerone"
	"github.com/sw33tLie/bbscope/pkg/scope"
)

var verboseLog bool

func SetVerbose(v bool) {
	verboseLog = v
}

func verbose(format string, a ...interface{}) {
	if verboseLog {
		// Using a simple print for client, as lipgloss is for styling the main output
		fmt.Printf("[VERBOSE] "+format+"\n", a...)
	}
}

// GetPrograms now uses the bbscope library and passes the required auth token and scope
func GetPrograms(authToken string, bbpOnly bool, scopeCategory string) ([]scope.ProgramData, error) {
	verbose("Fetching programs from HackerOne with bbpOnly=%t, scopeCategory=%s", bbpOnly, scopeCategory)
	// We pass the user's auth token, which is required by the H1 API.
	// All other parameters are set to sensible defaults for our use case.
	programs, err := hackerone.GetAllProgramsScope(authToken, bbpOnly, false, true, scopeCategory, true, 10, false, "", "", false)
	if err != nil {
		verbose("Error from hackerone.GetAllProgramsScope: %v", err)
		return nil, err
	}
	verbose("hackerone.GetAllProgramsScope returned %d programs before filtering", len(programs))

	// Post-filter to remove programs that have no in-scope items matching the category
	var filteredPrograms []scope.ProgramData
	for _, p := range programs {
		if len(p.InScope) > 0 {
			filteredPrograms = append(filteredPrograms, p)
		}
	}
	verbose("Filtered down to %d programs with at least one in-scope item", len(filteredPrograms))

	return filteredPrograms, nil
}
