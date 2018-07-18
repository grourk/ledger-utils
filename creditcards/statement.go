package creditcards

import (
	"fmt"
	"strconv"
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

func parseDateFromFilename(filename, delim string) (time.Time, error) {
	split := strings.Split(filename, delim)
	last := split[len(split)-1]
	dateStr := strings.Split(last, ".")[0]
	return time.Parse("20060102", dateStr)
}

func parseAmount(str string) (int, error) {
	if strings.HasPrefix(str, "$") {
		str = str[1:]
	}

	amountStr := strings.Split(str, ".")
	if len(amountStr) != 2 {
		return 0, fmt.Errorf("parsing amount '%s'", str)
	}

	dollars, err := strconv.Atoi(amountStr[0])
	if err != nil {
		return 0, fmt.Errorf("parsing dollars '%s'", amountStr[0])
	}

	neg := 1
	if dollars < 0 {
		neg = -1
		dollars = -dollars
	}

	cents, err := strconv.Atoi(amountStr[1])
	if err != nil {
		return 0, fmt.Errorf("parsing cents '%s'", amountStr[1])
	}

	if cents < 0 || cents > 99 {
		return 0, fmt.Errorf("invalid cents '%d'", cents)
	}

	return neg * (100*dollars + cents), nil
}
