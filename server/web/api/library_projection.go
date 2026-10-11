package api

import (
	"cmp"
	"math"
	"server/torr/state"
	"slices"
	"strings"
)

type LibraryResponse struct {
	Items        []state.TorrentStatus `json:"items"`
	Total        int                   `json:"total"`
	LibraryTotal int                   `json:"library_total"`
	Page         int                   `json:"page"`
	Limit        int                   `json:"limit"`
	Categories   []string              `json:"categories"`
	SampledAt    string                `json:"sampled_at"`
}

func projectLibrary(rows []state.TorrentStatus, query, category, order string, page, limit int) LibraryResponse {
	out := LibraryResponse{Items: []state.TorrentStatus{}, LibraryTotal: len(rows), Limit: limit, Categories: []string{}}
	categories := map[string]bool{}
	query = strings.ToLower(strings.TrimSpace(query))
	matched := make([]state.TorrentStatus, 0, len(rows))
	for _, row := range rows {
		if row.Category != "" {
			categories[row.Category] = true
		}
		if category == "uncategorized" && row.Category != "" {
			continue
		}
		if strings.HasPrefix(category, "category:") && row.Category != strings.TrimPrefix(category, "category:") {
			continue
		}
		if !strings.Contains(strings.ToLower(row.Title+" "+row.Name), query) {
			continue
		}
		matched = append(matched, row)
	}
	for category := range categories {
		out.Categories = append(out.Categories, category)
	}
	slices.Sort(out.Categories)
	slices.SortFunc(matched, func(a, b state.TorrentStatus) int {
		var result int
		switch order {
		case "title":
			result = cmp.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
		case "size":
			result = cmp.Compare(b.TorrentSize, a.TorrentSize)
		default:
			result = cmp.Compare(b.Timestamp, a.Timestamp)
		}
		if result == 0 {
			result = cmp.Compare(a.Hash, b.Hash)
		}
		return result
	})
	out.Total = len(matched)
	out.Page = max(1, min(page, max(1, int(math.Ceil(float64(out.Total)/float64(limit))))))
	start := (out.Page - 1) * limit
	out.Items = matched[start:min(start+limit, len(matched))]
	return out
}
