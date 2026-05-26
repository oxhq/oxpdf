package oxpdf

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePageRange parses a one-based page range into zero-based page indexes.
func ParsePageRange(spec string, pageCount int) ([]int, error) {
	if pageCount < 0 {
		return nil, fmt.Errorf("page count cannot be negative")
	}
	spec = strings.TrimSpace(spec)
	if spec == "" || strings.EqualFold(spec, "all") {
		pages := make([]int, pageCount)
		for i := range pages {
			pages[i] = i
		}
		return pages, nil
	}
	seen := make(map[int]bool)
	var pages []int
	for _, token := range strings.Split(spec, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			return nil, fmt.Errorf("empty page range token")
		}
		start, end, err := parsePageRangeToken(token)
		if err != nil {
			return nil, err
		}
		if start > end {
			return nil, fmt.Errorf("descending page range %q", token)
		}
		for page := start; page <= end; page++ {
			if page < 1 || page > pageCount {
				return nil, fmt.Errorf("page %d outside 1-%d", page, pageCount)
			}
			index := page - 1
			if !seen[index] {
				seen[index] = true
				pages = append(pages, index)
			}
		}
	}
	return pages, nil
}

func parsePageRangeToken(token string) (int, int, error) {
	parts := strings.Split(token, "-")
	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("malformed page range %q", token)
	}
	start, err := parseOneBasedPage(parts[0])
	if err != nil {
		return 0, 0, err
	}
	if len(parts) == 1 {
		return start, start, nil
	}
	end, err := parseOneBasedPage(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func parseOneBasedPage(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty page number")
	}
	page, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid page number %q", raw)
	}
	if page < 1 {
		return 0, fmt.Errorf("page numbers are one-based")
	}
	return page, nil
}
