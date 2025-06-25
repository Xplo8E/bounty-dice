package client

import (
	"github.com/sw33tLie/bbscope/pkg/platforms/hackerone"
	"github.com/sw33tLie/bbscope/pkg/scope"
)

// GetPrograms now uses the bbscope library and passes the required auth token and scope
func GetPrograms(authToken string, bbpOnly bool, scopeCategory string) ([]scope.ProgramData, error) {
	// We pass the user's auth token, which is required by the H1 API.
	// All other parameters are set to sensible defaults for our use case.
	programs, err := hackerone.GetAllProgramsScope(authToken, bbpOnly, false, true, scopeCategory, true, 10, false, "", "", false)
	if err != nil {
		return nil, err
	}

	// Post-filter to remove programs that have no in-scope items matching the category
	var filteredPrograms []scope.ProgramData
	for _, p := range programs {
		if len(p.InScope) > 0 {
			filteredPrograms = append(filteredPrograms, p)
		}
	}

	return filteredPrograms, nil
}