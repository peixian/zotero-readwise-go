package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

func readHighlightsFromCSV(filename string) ([]ReadwiseHighlight, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	records, err := reader.ReadAll()
	var highlights []ReadwiseHighlight
	for _, record := range records {
		if len(record) < 10 { // Updated length check
			continue
		}

		location, _ := strconv.Atoi(record[7])

		highlight := ReadwiseHighlight{
			Text:          record[0],
			Title:         record[1],
			Author:        record[2],
			SourceType:    record[3],
			Category:      record[4],
			Note:          record[5],
			HighlightedAt: record[6],
			Location:      location,
			LocationType:  record[8],
			SourceURL:     record[9],
		}
		highlights = append(highlights, highlight)
	}

	return highlights, nil
}

func writeHighlightsToCSV(highlights []ReadwiseHighlight, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{
		"text", "title", "author", "source_type", "category",
		"note", "highlighted_at", "location", "location_type", "source_url",
	}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, h := range highlights {
		if err := writer.Write([]string{
			h.Text,
			h.Title,
			h.Author,
			h.SourceType,
			h.Category,
			h.Note,
			h.HighlightedAt,
			strconv.Itoa(h.Location),
			h.LocationType,
			h.SourceURL,
		}); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}

func main() {
	var (
		inputCSV    = flag.String("input", "", "Input CSV file to send to Readwise")
		zoteroKey   = flag.String("zotero-key", "", "Zotero API key")
		readwiseKey = flag.String("readwise-key", "", "Readwise API key")
	)

	flag.Parse()

	if *readwiseKey == "" {
		fmt.Println("Error: Readwise API key is required")
		flag.Usage()
		os.Exit(1)
	}

	readwiseClient := NewReadwiseClient(*readwiseKey)
	var highlights []ReadwiseHighlight

	if *inputCSV != "" {
		// Mode 1: Read from CSV and send to Readwise
		fmt.Printf("Reading highlights from %s\n", *inputCSV)
		var err error
		highlights, err = readHighlightsFromCSV(*inputCSV)
		if err != nil {
			fmt.Printf("Error reading CSV: %v\n", err)
			return
		}
	} else {
		// Mode 2: Fetch from Zotero and send to Readwise
		if *zoteroKey == "" {
			fmt.Println("Error: Zotero API key is required when not using input CSV")
			flag.Usage()
			os.Exit(1)
		}

		zoteroClient := NewAPIClient(*zoteroKey)
		fmt.Println("Getting annotations from Zotero")

		annotations, err := zoteroClient.fetchAnnotations()
		if err != nil {
			fmt.Printf("Error fetching annotations: %v\n", err)
			return
		}

		fmt.Println("Converting annotations")
		highlights, err = zoteroClient.ConvertToReadwiseHighlights(annotations)
		if err != nil {
			fmt.Printf("Error converting annotations: %v\n", err)
			return
		}

		// Write to CSV as backup
		filename := fmt.Sprintf("highlights_backup_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
		if err := writeHighlightsToCSV(highlights, filename); err != nil {
			fmt.Printf("Error writing backup CSV: %v\n", err)
			return
		}
		fmt.Printf("Backup written to %s\n", filename)
	}

	fmt.Printf("Sending %d highlights to Readwise\n", len(highlights))
	if err := readwiseClient.SendHighlights(highlights); err != nil {
		fmt.Printf("Error sending highlights to Readwise: %v\n", err)
		return
	}

	fmt.Printf("Successfully sent %d highlights to Readwise\n", len(highlights))
}
