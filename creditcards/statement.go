package creditcards

import (
	"strings"
	"time"
)

type Entry struct {
	Description string
	Date        time.Time
	Amount      int
}

type Statement interface {
	Name() string
	Date() time.Time
	Total() int
	Entries() <-chan Entry
}

func parseDateFromFilename(filename string) (time.Time, error) {
	split := strings.Split(filename, "-")
	last := split[len(split)-1]
	dateStr := strings.Split(last, ".")[0]
	return time.Parse("20060102", dateStr)
}
