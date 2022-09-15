package ids

import (
	"bufio"
	_ "embed"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

type Denylist = []string

//go:embed denylist.txt
var defaultDenylistStr string
var defaultDenylist Denylist = strings.Split(defaultDenylistStr, "\n")

func LoadDenylist(filename string) (Denylist, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	denylist := make(Denylist, 0)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		denylist = append(denylist, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return denylist, nil
}

// IsAllowed returns false if the ID (normalised to lowecase) matches anything in the denylist, true otherwise.
func IsAllowed(id string, denylist Denylist) bool {
	normalisedID := strings.ToLower(id)

	// TODO: use (a pool of) goroutines to parallelise the search
	for _, bad := range denylist {
		if bad == "" {
			continue
		}

		if strings.Contains(normalisedID, bad) {
			log.Debug().
				Str("normalisedID", normalisedID).
				Str("bad", bad).
				Msg("IsAllowed: normalisedID matches bad word")
			return false
		}
	}
	return true
}
