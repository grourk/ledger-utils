package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/grourk/ledger-utils/creditcards"
)

var commandOpt string
var inputOpt string
var parserOpt string

var stdinReader *bufio.Reader

func main() {
	stdinReader = bufio.NewReader(os.Stdin)

	flag.StringVar(&commandOpt, "command", "cc", "one of: cc (credit card statement)")
	flag.StringVar(&inputOpt, "input", "", "input file")
	flag.StringVar(&parserOpt, "parser", "discover-card", "one of: discover-card")
	flag.Parse()

	var err error

	switch commandOpt {
	case "cc":
		err = handleStatement()
	default:
		err = fmt.Errorf("unknown command '%s'", commandOpt)
	}

	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
}

func formatAmount(amount int) string {
	dollars := amount / 100
	cents := amount % 100
	dollarsStr := humanize.Comma(int64(dollars))

	if cents == 0 {
		return fmt.Sprintf("$%s", dollarsStr)
	}

	if cents < 0 {
		cents = -cents
	}

	return fmt.Sprintf("$%s.%02d", dollarsStr, cents)
}

func readInput(def string, format string, a ...interface{}) (string, error) {
	fmt.Printf(format, a...)

	str, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}

	str = strings.TrimSpace(str)
	if str == "" {
		str = def
	}

	return str, nil
}

func handleStatement() error {
	statement, err := parse()
	if err != nil {
		return err
	}

	guesser, err := creditcards.NewGuesser()
	if err != nil {
		return err
	}
	defer guesser.Close()

	var output strings.Builder

	fmt.Fprintf(&output, "%s * %s Statement\n", statement.Date().Format("2006/1/2"), statement.Name())

	for entry := range statement.Entries() {
		amountFmt := formatAmount(entry.Amount)
		dateFmt := entry.Date.Format("2006/01/02")
		desc := entry.Description
		guess := guesser.MakeGuess(entry)

		category, err := readInput(guess, "%s %-8s %-75s (%s): ", dateFmt, amountFmt, desc, guess)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(category, "Expenses:") {
			return fmt.Errorf("invalid category '%s'", category)
		}

		guesser.ConfirmGuess(entry, category)

		fmt.Fprintf(&output, "  %-55s %-12s; %s - %s\n", category, amountFmt, dateFmt, desc)
	}

	category := fmt.Sprintf("Liabilities:%s", statement.Name())
	totalFmt := formatAmount(-statement.Total())
	fmt.Fprintf(&output, "  %-55s %s\n", category, totalFmt)

	fmt.Println()
	fmt.Println(output.String())

	return nil
}

func parse() (creditcards.Statement, error) {
	var statement creditcards.Statement

	switch parserOpt {
	case "discover-card":
		var err error
		statement, err = creditcards.NewDiscoverCardStatement(inputOpt)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown parser '%s'", parserOpt)
	}

	var sum = 0
	for entry := range statement.Entries() {
		sum += entry.Amount
	}

	var err error
	if sum != statement.Total() {
		err = fmt.Errorf("sum of amounts %d does not match total %d", sum, statement.Total())
	}

	return statement, err
}
