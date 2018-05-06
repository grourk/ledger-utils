package creditcards

import "time"

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
