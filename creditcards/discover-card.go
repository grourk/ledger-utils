package creditcards

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type DiscoverCardStatement struct {
	date    time.Time
	total   int
	entries []Entry
}

func parseDate(filename string) (time.Time, error) {
	split := strings.Split(filename, "-")
	last := split[len(split)-1]
	dateStr := strings.Split(last, ".")[0]
	return time.Parse("20060102", dateStr)
}

func NewDiscoverCardStatement(filename string) (*DiscoverCardStatement, error) {
	in, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	defer in.Close()

	statementDate, err := parseDate(filename)
	if err != nil {
		return nil, fmt.Errorf("parsing date from filename '%s': %v", filename, err)
	}

	r := csv.NewReader(in)
	entries := make([]Entry, 0, 100)
	total := 0
	first := true

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading csv: %v", err)
		}

		if first {
			first = false
			continue
		}

		// Trans. Date,Post Date,Description,Amount,Category
		desc := record[2]
		cat := record[4]

		if cat == "Payments and Credits" && strings.HasPrefix(desc, "DIRECTPAY ") {
			continue
		}

		date, err := time.Parse("01/02/2006", record[0])
		if err != nil {
			return nil, fmt.Errorf("parsing date '%s': %v", record[0], err)
		}

		amountStr := strings.Split(record[3], ".")
		if len(amountStr) != 2 {
			return nil, fmt.Errorf("parsing amount '%s'", record[3])
		}

		dollars, err := strconv.Atoi(amountStr[0])
		if err != nil {
			return nil, fmt.Errorf("parsing dollars '%s'", amountStr[0])
		}

		cents, err := strconv.Atoi(amountStr[1])
		if err != nil {
			return nil, fmt.Errorf("parsing cents '%s'", amountStr[1])
		}

		if cents < 0 || cents > 99 {
			return nil, fmt.Errorf("invalid cents '%d'", cents)
		}

		amount := 100*dollars + cents

		total += amount

		entries = append(entries, Entry{desc, date, amount})
	}

	return &DiscoverCardStatement{statementDate, total, entries}, nil
}

func (*DiscoverCardStatement) Name() string {
	return "Discover Card"
}

func (s *DiscoverCardStatement) Date() time.Time {
	return s.date
}

func (s *DiscoverCardStatement) Total() int {
	return s.total
}

func (s *DiscoverCardStatement) Entries() <-chan Entry {
	ch := make(chan Entry)

	go func() {
		for _, e := range s.entries {
			ch <- e
		}
		close(ch)
	}()

	return ch
}
