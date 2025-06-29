package random

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/sw33tLie/bbscope/pkg/scope"
)

var verboseLog bool

func SetVerbose(v bool) {
	verboseLog = v
}

func verbose(format string, a ...interface{}) {
	if verboseLog {
		fmt.Printf("[VERBOSE] "+format+"\n", a...)
	}
}

func Select(programs []scope.ProgramData) (scope.ProgramData, error) {
	if len(programs) == 0 {
		verbose("Cannot select from an empty list of programs.")
		return scope.ProgramData{}, nil
	}
	verbose("Selecting one random program from a list of %d.", len(programs))

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(programs))))
	if err != nil {
		verbose("Error generating random number: %v", err)
		return scope.ProgramData{}, err
	}

	selectedIndex := n.Int64()
	selectedProgram := programs[selectedIndex]
	verbose("Randomly selected index %d, which corresponds to program: %s", selectedIndex, selectedProgram.Url)

	return selectedProgram, nil
}