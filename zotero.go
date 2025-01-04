package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type FirstLevelResponse []AnnotationItem

type AnnotationItem struct {
	Key     string      `json:"key"`
	Version int         `json:"version"`
	Library Library     `json:"library"`
	Links   LinksObject `json:"links"`
	Data    ItemData    `json:"data"`
}

type Library struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type LinksObject struct {
	Self      Link `json:"self"`
	Alternate Link `json:"alternate"`
	Up        Link `json:"up"`
}

type Link struct {
	Href string `json:"href"`
	Type string `json:"type,omitempty"`
}

// type ItemData struct {
// 	Key             string    `json:"key"`
// 	Version         int       `json:"version"`
// 	ParentItem      string    `json:"parentItem"`
// 	ItemType        string    `json:"itemType"`
// 	Title           string    `json:"title"`
// 	Creators        []Creator `json:"creators"`
// 	Date            string    `json:"date"`
// 	AnnotationType  string    `json:"annotationType"`
// 	AnnotationText  string    `json:"annotationText"`
// 	AnnotationColor string    `json:"annotationColor"`
// 	DateAdded       string    `json:"dateAdded"`
// 	DateModified    string    `json:"dateModified"`
// }

type ItemData struct {
	Key                 string    `json:"key"`
	Version             int       `json:"version"`
	ParentItem          string    `json:"parentItem"`
	ItemType            string    `json:"itemType"`
	Title               string    `json:"title"`
	Creators            []Creator `json:"creators"`
	Date                string    `json:"date"`
	URL                 string    `json:"url"`
	DOI                 string    `json:"DOI,omitempty"` // Also grab DOI if available
	AnnotationType      string    `json:"annotationType"`
	AnnotationText      string    `json:"annotationText"`
	AnnotationColor     string    `json:"annotationColor"`
	AnnotationPageLabel string    `json:"annotationPageLabel"`
	DateAdded           string    `json:"dateAdded"`
	DateModified        string    `json:"dateModified"`
}

type Creator struct {
	CreatorType string `json:"creatorType"`
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Name        string `json:"name,omitempty"`      // For single-field institutional authors
	FieldMode   int    `json:"fieldMode,omitempty"` // 0 for two-field (person), 1 for single-field (institution)
}

type APIClient struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

// type ParentDetails struct {
// 	Title   string
// 	Authors []Creator
// 	Date    string
// }

type ParentDetails struct {
	Title    string
	Authors  []Creator
	Date     string
	ItemType string
	URL      string
}

func mapZoteroTypeToReadwise(zoteroType string) (sourceType, category string) {
	// Default values
	sourceType = "zotero"
	category = "articles"

	// Types that should be categorized as books in Readwise
	bookTypes := map[string]bool{
		"book":         true,
		"thesis":       true,
		"manuscript":   true,
		"bookSection":  true,
		"monograph":    true,
		"dissertation": true,
	}

	// Types that should be categorized as articles
	articleTypes := map[string]bool{
		"journalArticle":   true,
		"preprint":         true,
		"report":           true,
		"conferencePaper":  true,
		"magazineArticle":  true,
		"newspaperArticle": true,
		"webpage":          true,
		"blogPost":         true,
		"document":         true,
	}

	if bookTypes[zoteroType] {
		category = "books"
	} else if articleTypes[zoteroType] {
		category = "articles"
	}

	return sourceType, category
}

func formatCreator(c Creator) string {
	switch {
	// Handle institutional authors (single field mode)
	case c.FieldMode == 1 || (c.Name != "" && c.FirstName == "" && c.LastName == ""):
		return c.Name

	// Handle personal authors (two field mode)
	case c.FirstName != "" || c.LastName != "":
		// Handle cases where only one name part exists
		if c.FirstName == "" {
			return c.LastName
		}
		if c.LastName == "" {
			return c.FirstName
		}
		return fmt.Sprintf("%s, %s", c.LastName, c.FirstName)

	default:
		return ""
	}
}

func formatCreators(creators []Creator) string {
	if len(creators) == 0 {
		return ""
	}

	// Group creators by type
	creatorsByType := make(map[string][]Creator)
	for _, c := range creators {
		creatorsByType[c.CreatorType] = append(creatorsByType[c.CreatorType], c)
	}

	var result []string

	// Process authors first if they exist
	if authors, exists := creatorsByType["author"]; exists {
		var authorNames []string
		for _, author := range authors {
			if formatted := formatCreator(author); formatted != "" {
				authorNames = append(authorNames, formatted)
			}
		}
		if len(authorNames) > 0 {
			result = append(result, strings.Join(authorNames, "; "))
		}
	}

	// Process other creator types
	for creatorType, creators := range creatorsByType {
		if creatorType == "author" {
			continue
		}

		var names []string
		for _, creator := range creators {
			if formatted := formatCreator(creator); formatted != "" {
				names = append(names, formatted)
			}
		}

		if len(names) > 0 {
			result = append(result, fmt.Sprintf("%s: %s",
				humanizeCreatorType(creatorType),
				strings.Join(names, "; ")))
		}
	}

	return strings.Join(result, " | ")
}

func humanizeCreatorType(creatorType string) string {
	// Map of special cases
	special := map[string]string{
		"seriesEditor":  "Series Editor",
		"bookAuthor":    "Book Author",
		"commenter":     "Commentator",
		"scriptwriter":  "Scriptwriter",
		"wordsBy":       "Words By",
		"attorneyAgent": "Attorney/Agent",
	}

	if special, ok := special[creatorType]; ok {
		return special
	}

	// Default case: capitalize first letter
	if creatorType == "" {
		return ""
	}
	return strings.ToUpper(creatorType[:1]) + creatorType[1:]
}

func NewAPIClient(apiKey string) *APIClient {
	return &APIClient{
		client:  &http.Client{},
		apiKey:  apiKey,
		baseURL: "https://api.zotero.org/users/5466518/items",
	}
}

func (c *APIClient) makeRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Add("Zotero-API-Version", "3")

	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("making request: %w", err)
		}
		defer resp.Body.Close()

		// Handle backoff header if present
		if backoffStr := resp.Header.Get("Backoff"); backoffStr != "" {
			backoffSeconds, err := strconv.Atoi(backoffStr)
			if err == nil && backoffSeconds > 0 {
				time.Sleep(time.Duration(backoffSeconds) * time.Second)
			}
			fmt.Printf("backing off...")
		}

		// Handle rate limiting and server maintenance
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			retryAfterStr := resp.Header.Get("Retry-After")
			retryAfter, err := strconv.Atoi(retryAfterStr)
			if err != nil {
				retryAfter = 60 // Default to 60 seconds if header is missing or invalid
			}

			if attempt < maxRetries {
				time.Sleep(time.Duration(retryAfter) * time.Second)
				fmt.Printf("waiting to retry")
				continue
			}
			return nil, fmt.Errorf("max retries exceeded after receiving %d status", resp.StatusCode)
		}

		// Handle other error status codes
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	}

	return nil, fmt.Errorf("max retries exceeded")
}

func (c *APIClient) fetchAnnotations() ([]AnnotationItem, error) {
	var allItems []AnnotationItem
	start := 0
	limit := 100

	for {
		url := fmt.Sprintf("%s?itemType=annotation&sort=dateModified&direction=asc&start=%d&limit=%d",
			c.baseURL, start, limit)

		body, err := c.makeRequest(url)
		if err != nil {
			return nil, fmt.Errorf("fetching annotations: %w", err)
		}

		var items FirstLevelResponse
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("parsing JSON: %w", err)
		}

		allItems = append(allItems, items...)

		if len(items) < limit {
			break
		}
		start += limit

		fmt.Printf("current annotations count: %d\n", len(allItems))
		// Add a small delay between requests to be nice to the API
		time.Sleep(10 * time.Millisecond)
	}

	return allItems, nil
}

func (c *APIClient) fetchParentDetails(item AnnotationItem) (*ParentDetails, error) {
	// First level - get immediate parent (usually attachment)
	parent, err := c.makeRequest(item.Links.Up.Href)
	if err != nil {
		return nil, fmt.Errorf("fetching immediate parent: %w", err)
	}

	var parentItem AnnotationItem
	if err := json.Unmarshal(parent, &parentItem); err != nil {
		return nil, fmt.Errorf("parsing parent JSON: %w", err)
	}

	// If parent is attachment type, fetch grandparent
	if parentItem.Data.ItemType == "attachment" {
		grandparent, err := c.makeRequest(parentItem.Links.Up.Href)
		if err != nil {
			return nil, fmt.Errorf("fetching grandparent: %w", err)
		}

		var grandparentItem AnnotationItem
		if err := json.Unmarshal(grandparent, &grandparentItem); err != nil {
			return nil, fmt.Errorf("parsing grandparent JSON: %w", err)
		}

		// Construct URL, preferring DOI if available
		url := grandparentItem.Data.URL
		if grandparentItem.Data.DOI != "" {
			url = fmt.Sprintf("https://doi.org/%s", grandparentItem.Data.DOI)
		}

		return &ParentDetails{
			Title:    grandparentItem.Data.Title,
			Authors:  grandparentItem.Data.Creators,
			Date:     grandparentItem.Data.Date,
			ItemType: grandparentItem.Data.ItemType,
			URL:      url,
		}, nil
	}

	// If parent is not attachment, use parent details
	url := parentItem.Data.URL
	if parentItem.Data.DOI != "" {
		url = fmt.Sprintf("https://doi.org/%s", parentItem.Data.DOI)
	}

	return &ParentDetails{
		Title:    parentItem.Data.Title,
		Authors:  parentItem.Data.Creators,
		Date:     parentItem.Data.Date,
		ItemType: parentItem.Data.ItemType,
		URL:      url,
	}, nil
}

// func (c *APIClient) fetchParentDetails(item AnnotationItem) (*ParentDetails, error) {
// 	// First level - get immediate parent (usually attachment)
// 	parent, err := c.makeRequest(item.Links.Up.Href)
// 	if err != nil {
// 		return nil, fmt.Errorf("fetching immediate parent: %w", err)
// 	}

// 	var parentItem AnnotationItem
// 	if err := json.Unmarshal(parent, &parentItem); err != nil {
// 		return nil, fmt.Errorf("parsing parent JSON: %w", err)
// 	}

// 	// If parent is attachment type, fetch grandparent
// 	if parentItem.Data.ItemType == "attachment" {
// 		grandparent, err := c.makeRequest(parentItem.Links.Up.Href)
// 		if err != nil {
// 			return nil, fmt.Errorf("fetching grandparent: %w", err)
// 		}

// 		var grandparentItem AnnotationItem
// 		if err := json.Unmarshal(grandparent, &grandparentItem); err != nil {
// 			return nil, fmt.Errorf("parsing grandparent JSON: %w", err)
// 		}

// 		return &ParentDetails{
// 			Title:   grandparentItem.Data.Title,
// 			Authors: grandparentItem.Data.Creators,
// 			Date:    grandparentItem.Data.Date,
// 		}, nil
// 	}

// 	// If parent is not attachment, use parent details
// 	return &ParentDetails{
// 		Title:   parentItem.Data.Title,
// 		Authors: parentItem.Data.Creators,
// 		Date:    parentItem.Data.Date,
// 	}, nil
// }

func (c *APIClient) processAnnotations(annotations []AnnotationItem) error {
	fmt.Println("annotation_text,title,authors,date")
	var skipped int

	for _, annotation := range annotations {
		// Skip empty annotations
		if strings.TrimSpace(annotation.Data.AnnotationText) == "" {
			skipped++
			fmt.Fprintf(os.Stderr, "Skipping empty annotation with key: %s\n", annotation.Key)
			continue
		}

		details, err := c.fetchParentDetails(annotation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching parent details for %s: %v\n", annotation.Key, err)
			continue
		}

		// Skip if we couldn't get valid parent details
		if details.Title == "" {
			skipped++
			fmt.Fprintf(os.Stderr, "Skipping annotation %s: empty parent title\n", annotation.Key)
			continue
		}

		time.Sleep(10 * time.Millisecond)

		fmt.Printf("%s,%s,%s,%s\n",
			strings.ReplaceAll(annotation.Data.AnnotationText, ",", ""),
			strings.ReplaceAll(details.Title, ",", ""),
			formatCreators(details.Authors),
			details.Date)
	}

	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "Total annotations skipped: %d\n", skipped)
	}

	return nil
}

// func (c *APIClient) ConvertToReadwiseHighlights(annotations []AnnotationItem) ([]ReadwiseHighlight, error) {
//     var highlights []ReadwiseHighlight

//     for _, annotation := range annotations {
//         if strings.TrimSpace(annotation.Data.AnnotationText) == "" {
//             continue
//         }

//         details, err := c.fetchParentDetails(annotation)
//         if err != nil {
//             fmt.Fprintf(os.Stderr, "Error fetching parent details for %s: %v\n", annotation.Key, err)
//             continue
//         }

//         author := formatCreators(details.Authors)
//         if author == "" {
//             author = "Unknown Author"
//         }

//         sourceType, category := mapZoteroTypeToReadwise(details.ItemType)

//         highlight := ReadwiseHighlight{
//             Text:          annotation.Data.AnnotationText,
//             Title:         details.Title,
//             Author:        author,
//             SourceType:    sourceType,
//             Category:      category,
//             HighlightedAt: annotation.Data.DateAdded,
//         }

//         if annotation.Data.AnnotationPageLabel != "" {
//             if pageNum, err := strconv.Atoi(annotation.Data.AnnotationPageLabel); err == nil {
//                 highlight.Location = pageNum
//                 highlight.LocationType = "page"
//             }
//         }

//         highlights = append(highlights, highlight)
//     }

//     return highlights, nil
// }

func (c *APIClient) ConvertToReadwiseHighlights(annotations []AnnotationItem) ([]ReadwiseHighlight, error) {
	var highlights []ReadwiseHighlight

	for _, annotation := range annotations {
		if strings.TrimSpace(annotation.Data.AnnotationText) == "" {
			continue
		}

		details, err := c.fetchParentDetails(annotation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching parent details for %s: %v\n", annotation.Key, err)
			continue
		}

		author := formatCreators(details.Authors)
		if author == "" {
			author = "Unknown Author"
		}

		sourceType, category := mapZoteroTypeToReadwise(details.ItemType)

		highlight := ReadwiseHighlight{
			Text:          annotation.Data.AnnotationText,
			Title:         details.Title,
			Author:        author,
			SourceType:    sourceType,
			Category:      category,
			HighlightedAt: annotation.Data.DateAdded,
			SourceURL:     details.URL,
		}

		if annotation.Data.AnnotationPageLabel != "" {
			if pageNum, err := strconv.Atoi(annotation.Data.AnnotationPageLabel); err == nil {
				highlight.Location = pageNum
				highlight.LocationType = "page"
			}
		}

		highlights = append(highlights, highlight)
	}

	return highlights, nil
}
