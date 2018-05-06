package creditcards

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"

	"github.com/schollz/closestmatch"
)

type Guesser struct {
	// TODO: also ability to read in Amazon purchases and match
	filename string
	records  map[string]string // normalized description -> category
	matcher  *closestmatch.ClosestMatch
}

func normalize(str string) string {
	return strings.ToUpper(strings.TrimSpace(str))
}

func getFilename() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return usr.HomeDir + "/.ledger-utils-guesses", nil
}

func NewGuesser() (*Guesser, error) {
	records := make(map[string]string)
	filename, err := getFilename()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filename)

	if err == nil {
		defer f.Close()

		reader := bufio.NewReader(f)

		for {
			line, err := reader.ReadString('\n')

			if err == io.EOF {
				break
			}

			if err != nil {
				continue
			}

			split := strings.Split(line, " -> ")

			if len(split) == 2 {
				records[normalize(split[0])] = strings.TrimSpace(split[1])
			}
		}
	}

	words := make([]string, 0, len(records))

	for k := range records {
		words = append(words, k)
	}

	bagSizes := []int{2, 3, 4} // TODO ??
	matcher := closestmatch.New(words, bagSizes)

	return &Guesser{filename, records, matcher}, nil
}

func (g *Guesser) Close() {
	f, err := os.Create(g.filename)

	if err != nil {
		return
	}

	defer f.Close()

	for desc, cat := range g.records {
		fmt.Fprintf(f, "%s -> %s\n", desc, cat)
	}
}

func (g *Guesser) MakeGuess(entry Entry) string {
	norm := normalize(entry.Description)
	existing := g.records[norm]

	if existing != "" {
		return existing
	}

	closest := g.matcher.Closest(norm)

	return g.records[closest]
}

func (g *Guesser) ConfirmGuess(entry Entry, category string) {
	norm := normalize(entry.Description)
	cat := strings.TrimSpace(category)
	g.records[norm] = cat
}
