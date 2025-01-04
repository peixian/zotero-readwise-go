package main

import (
	"fmt"
	"os"
	"encoding/csv"
	"time"
)



func writeHighlightsToCSV(highlights []ReadwiseHighlight, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"text", "title", "author", "source_type", "category", "note", "highlighted_at"}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write highlights
	for _, h := range highlights {
		if err := writer.Write([]string{
			h.Text,
			h.Title,
			h.Author,
			h.SourceType,
			h.Category,
			h.Note,
			h.HighlightedAt,
		}); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}

func main() {
	zoteroClient := NewAPIClient("YOUR_ZOTERO_API_KEY")
	readwiseClient := NewReadwiseClient("j4z9gXgHU8Mb6FjMo9atHpcwdgXuYr1RwXVNK6AvTMJTmflGDU")

	fmt.Printf("Getting annotations")

	annotations, err := zoteroClient.fetchAnnotations()
	if err != nil {
		fmt.Printf("Error fetching annotations: %v\n", err)
		return
	}

	fmt.Printf("Converting annotations")

	highlights, err := zoteroClient.ConvertToReadwiseHighlights(annotations)
	if err != nil {
		fmt.Printf("Error converting annotations: %v\n", err)
		return
	}

	// Write to CSV before sending to Readwise
	filename := fmt.Sprintf("highlights_backup_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	if err := writeHighlightsToCSV(highlights, filename); err != nil {
		fmt.Printf("Error writing backup CSV: %v\n", err)
		return
	}

	fmt.Printf("Backup written to %s\n", filename)
	fmt.Printf("Sending to readwise")


	if err := readwiseClient.SendHighlights(highlights); err != nil {
		fmt.Printf("Error sending highlights to Readwise: %v\n", err)
		return
	}

	fmt.Printf("Successfully sent %d highlights to Readwise\n", len(highlights))
}
