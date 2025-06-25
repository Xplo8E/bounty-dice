package random

import (
	"crypto/rand"
	"math/big"

	"github.com/sw33tLie/bbscope/pkg/scope"
)

func Select(programs []scope.ProgramData) (scope.ProgramData, error) {
	if len(programs) == 0 {
		return scope.ProgramData{}, nil
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(programs))))
	if err != nil {
		return scope.ProgramData{}, err
	}

	return programs[n.Int64()], nil
}
