package creditcards

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type DiscoverCardStatement struct {
	date    time.Time
	total   int
	entries []Entry
}

func NewDiscoverCardStatement(filename string) (*DiscoverCardStatement, error) {
	in, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	statementDate, err := parseDateFromFilename(filename, "-")
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

		amount, err := parseAmount(record[3])
		if err != nil {
			return nil, err
		}

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
