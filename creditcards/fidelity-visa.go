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

/*
 Manual steps:
 - Open statement and get date range
 - Download transactions from Fidelity for date range
 - Rename file: mv ~/Downloads/download.csv ~/Downloads/Fidelity-Visa-Statement-20180716.csv
 - Ensure output total matches statement
*/

type FidelityVisaStatement struct {
	date    time.Time
	total   int
	entries []Entry
}

func NewFidelityVisaStatement(filename string) (*FidelityVisaStatement, error) {
	in, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	defer in.Close()

	statementDate, err := parseDateFromFilename(filename)
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

		// Date,Transaction,Name,Memo,Amount
		trans := record[1]
		if trans != "DEBIT" {
			continue
		}

		desc := record[2]

		date, err := time.Parse("1/2/2006", record[0])
		if err != nil {
			return nil, fmt.Errorf("parsing date '%s': %v", record[0], err)
		}

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

	return &FidelityVisaStatement{statementDate, total, entries}, nil
}

func (*FidelityVisaStatement) Name() string {
	return "Fidelity Visa"
}

func (s *FidelityVisaStatement) Date() time.Time {
	return s.date
}

func (s *FidelityVisaStatement) Total() int {
	return s.total
}

func (s *FidelityVisaStatement) Entries() <-chan Entry {
	ch := make(chan Entry)

	go func() {
		for _, e := range s.entries {
			ch <- e
		}
		close(ch)
	}()

	return ch
}
