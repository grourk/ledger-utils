package creditcards

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/schollz/closestmatch"
)

type Guesser struct {
	filename string
	records  map[string]string // normalized description -> category
	matcher  *closestmatch.ClosestMatch
	orders   []*amazonOrder
}

type amazonOrder struct {
	title        string
	pricePerUnit int
	quantity     int
	shipmentDate time.Time
	subtotalTax  int
	total        int
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

func parseAmazonOrders(filename string) ([]*amazonOrder, error) {
	if strings.HasPrefix(filename, "~/") {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("getting current user: %v", err)
		}
		filename = filepath.Join(usr.HomeDir, filename[2:])
	}

	in, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	r := csv.NewReader(in)
	orders := make([]*amazonOrder, 0, 100)
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

		// Order Date,Order ID,Title,Category,ASIN/ISBN,UNSPSC Code,Website,Release Date,Condition,Seller,Seller Credentials,List Price Per Unit,Purchase Price Per Unit,Quantity,Payment Instrument Type,Purchase Order Number,PO Line Number,Ordering Customer Email,Shipment Date,Shipping Address Name,Shipping Address Street 1,Shipping Address Street 2,Shipping Address City,Shipping Address State,Shipping Address Zip,Order Status,Carrier Name & Tracking Number,Item Subtotal,Item Subtotal Tax,Item Total,Tax Exemption Applied,Tax Exemption Type,Exemption Opt-Out,Buyer Name,Currency,Group Name

		if record[25] != "Shipped" {
			// Skip orders that aren't shipped yet
			continue
		}

		title := record[2]

		pricePerUnit, err := parseAmount(record[12])
		if err != nil {
			return nil, err
		}

		quantity, err := strconv.Atoi(record[13])
		if err != nil {
			return nil, fmt.Errorf("parsing quantity '%s'", record[13])
		}

		shipmentDate, err := time.Parse("01/02/06", record[18])
		if err != nil {
			return nil, fmt.Errorf("parsing shipment date '%s': %v", record[18], err)
		}

		subtotalTax, err := parseAmount(record[28])
		if err != nil {
			return nil, err
		}

		total, err := parseAmount(record[29])
		if err != nil {
			return nil, err
		}

		order := &amazonOrder{title, pricePerUnit, quantity, shipmentDate, subtotalTax, total}
		orders = append(orders, order)
	}

	return orders, nil
}

func NewGuesser(amazonOrdersFilenames []string) (*Guesser, error) {
	var orders []*amazonOrder
	for _, fn := range amazonOrdersFilenames {
		os, err := parseAmazonOrders(fn)
		if err != nil {
			return nil, fmt.Errorf("parsing amazon orders from %s: %v", fn, err)
		}
		orders = append(orders, os...)
	}

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

	return &Guesser{filename, records, matcher, orders}, nil
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

func (g *Guesser) matchAmazonOrder(entry Entry) *amazonOrder {
	// Choose candidate with lowest price delta
	var candidate *amazonOrder
	var minDelta int

	for _, order := range g.orders {
		// Match orders that were chargee on shipment date or day after
		if order.shipmentDate == entry.Date || order.shipmentDate.AddDate(0, 0, 1) == entry.Date {
			if order.total == entry.Amount || order.total-1 == entry.Amount || order.total+1 == entry.Amount {
				// And matches total give or take a cent
				delta := order.total - entry.Amount
				if delta < 0 {
					delta = -delta
				}
				if candidate == nil || delta < minDelta {
					candidate = order
					minDelta = delta
				}
			} else {
				// Or matches up to 5-25% discount off item subtotal (i.e., subscribe & save)
				low := int(float64(order.pricePerUnit*order.quantity)*0.75) + order.subtotalTax
				high := int(float64(order.pricePerUnit*order.quantity)*0.95) + order.subtotalTax + 1
				if low <= entry.Amount && entry.Amount <= high {
					delta := entry.Amount - low
					if (high - entry.Amount) < delta {
						delta = high - entry.Amount
					}
					if candidate == nil || delta < minDelta {
						candidate = order
						minDelta = delta
					}
				}
			}
		}
	}

	return candidate
}

func (g *Guesser) MakeGuess(entry Entry) (string, string) {
	norm := normalize(entry.Description)

	var title string

	if strings.Contains(norm, "AMAZON") {
		order := g.matchAmazonOrder(entry)
		if order != nil {
			title = order.title
			norm = normalize(title)
		}
	}

	existing := g.records[norm]
	if existing != "" {
		return existing, title
	}

	closest := g.matcher.Closest(norm)
	return g.records[closest], title
}

func (g *Guesser) ConfirmGuess(entry Entry, order, category string) {
	var norm string
	if order != "" {
		norm = normalize(order)
	} else {
		norm = normalize(entry.Description)
	}
	cat := strings.TrimSpace(category)
	g.records[norm] = cat
}
