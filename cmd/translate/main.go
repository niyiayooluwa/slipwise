// Command translate is a local diagnostic CLI tool for developers.
// It reads raw SportyBet JSON payloads dumped from the browser/scraper, runs them
// through our internal translator, and outputs clean normalized matches and selections.
//
// Why this exists:
// SportyBet's JSON format changes, contains weird nested keys, and has dozens of market variants.
// Instead of debugging through API requests and database rows, developers can pipe raw JSON
// files directly through this tool to see if our translator maps every market properly.
//
// Usage:
//
//	go run ./cmd/translate -in ai/sportybet_samples/codeShare.json
//	go run ./cmd/translate -in ai/sportybet_samples/last_sample.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"slipwise/internal/betting/domain"
	"slipwise/internal/betting/provider/sportybet"
)

const defaultInPath = "ai/sportybet_samples/codeShare.json"

func main() {
	inPath := flag.String("in", defaultInPath, "path to the SportyBet JSON payload file")
	flag.Parse()

	raw, err := os.ReadFile(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", *inPath, err)
		os.Exit(1)
	}

	var payload sportybet.SportyBetPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	matches, selections := sportybet.TranslateSportyBet(payload)

	fmt.Printf("matches: %d, selections: %d\n\n", len(matches), len(selections))

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	fmt.Println("-- matches --")
	if err := enc.Encode(matches); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding matches: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n-- booking_selections --")
	if err := enc.Encode(selections); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding selections: %v\n", err)
		os.Exit(1)
	}

	// Flag any selection the translator couldn't map, since these would
	// silently sit as UNKNOWN market type in the DB otherwise.
	var unmapped []domain.BookingSelection
	for _, s := range selections {
		if s.MarketType == "UNKNOWN" || s.Selection == "" {
			unmapped = append(unmapped, s)
		}
	}
	if len(unmapped) > 0 {
		fmt.Fprintf(os.Stderr, "\nwarning: %d selection(s) could not be fully mapped:\n", len(unmapped))
		enc.Encode(unmapped)
	}
}
