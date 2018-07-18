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
var amazonOrdersFilenamesOpt []string

var stdinReader *bufio.Reader

func main() {
	stdinReader = bufio.NewReader(os.Stdin)

	var amazonOrdersFilenames string
	flag.StringVar(&commandOpt, "command", "cc", "one of: cc (credit card statement)")
	flag.StringVar(&inputOpt, "input", "", "input file")
	flag.StringVar(&parserOpt, "parser", "discover-card", "one of: discover-card, fidelity-visa, chase-visa, amazon-visa")
	flag.StringVar(&amazonOrdersFilenames, "orders", "", "amazon orders files")
	flag.Parse()

	if amazonOrdersFilenames != "" {
		amazonOrdersFilenamesOpt = strings.Split(amazonOrdersFilenames, ",")
	}

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

	guesser, err := creditcards.NewGuesser(amazonOrdersFilenamesOpt)
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
		guess, order := guesser.MakeGuess(entry)

		if order != "" {
			desc = order
		}

		if len(desc) > 85 {
			desc = desc[0:85]
		}

		category, err := readInput(guess, "%s %-8s %-85s (%s): ", dateFmt, amountFmt, desc, guess)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(category, "Expenses:") {
			return fmt.Errorf("invalid category '%s'", category)
		}

		guesser.ConfirmGuess(entry, order, category)

		if len(desc) > 70 {
			desc = desc[0:70]
		}

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
	var err error

	switch parserOpt {
	case "discover-card":
		statement, err = creditcards.NewDiscoverCardStatement(inputOpt)
		if err != nil {
			return nil, err
		}
	case "fidelity-visa":
		statement, err = creditcards.NewFidelityVisaStatement(inputOpt)
		if err != nil {
			return nil, err
		}
	case "chase-visa":
		statement, err = creditcards.NewChaseVisaStatement(inputOpt)
		if err != nil {
			return nil, err
		}
	case "amazon-visa":
		statement, err = creditcards.NewAmazonVisaStatement(inputOpt)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown parser '%s'", parserOpt)
	}

	sum := 0
	for entry := range statement.Entries() {
		sum += entry.Amount
	}

	if sum != statement.Total() {
		err = fmt.Errorf("sum of amounts %d does not match total %d", sum, statement.Total())
	}

	return statement, err
}
