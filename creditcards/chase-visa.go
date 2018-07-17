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

type ChaseVisaStatement struct {
	name    string
	date    time.Time
	total   int
	entries []Entry
}

func NewChaseVisaStatement(filename string) (*ChaseVisaStatement, error) {
	return newChaseVisaStatement(filename, "Chase Visa")
}

func NewAmazonVisaStatement(filename string) (*ChaseVisaStatement, error) {
	return newChaseVisaStatement(filename, "Amazon Visa")
}

func newChaseVisaStatement(filename, name string) (*ChaseVisaStatement, error) {
	in, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	defer in.Close()

	statementDate, err := parseDateFromFilename(filename, "_")
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

		// Type,Trans Date,Post Date,Description,Amount
		typ := record[0]
		if typ != "Sale" {
			continue
		}

		date, err := time.Parse("01/02/2006", record[1])
		if err != nil {
			return nil, fmt.Errorf("parsing date '%s': %v", record[1], err)
		}

		desc := record[3]

		amountStr := strings.Split(record[4], ".")
		if len(amountStr) != 2 {
			return nil, fmt.Errorf("parsing amount '%s'", record[4])
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

		amount := -100*dollars + cents

		total += amount

		entries = append(entries, Entry{desc, date, amount})
	}

	return &ChaseVisaStatement{name, statementDate, total, entries}, nil
}

func (s *ChaseVisaStatement) Name() string {
	return s.name
}

func (s *ChaseVisaStatement) Date() time.Time {
	return s.date
}

func (s *ChaseVisaStatement) Total() int {
	return s.total
}

func (s *ChaseVisaStatement) Entries() <-chan Entry {
	ch := make(chan Entry)

	go func() {
		for _, e := range s.entries {
			ch <- e
		}
		close(ch)
	}()

	return ch
}
