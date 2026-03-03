package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	searchconsoleapi "google.golang.org/api/searchconsole/v1"

	"github.com/steipete/gogcli/internal/googleapi"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

var newSearchConsoleService = googleapi.NewSearchConsole

type SearchConsoleCmd struct {
	Sites SearchConsoleSitesCmd `cmd:"" name:"sites" aliases:"list,ls" default:"withargs" help:"List Search Console sites"`
	Query SearchConsoleQueryCmd `cmd:"" name:"query" aliases:"report" help:"Run a Search Analytics query"`
}

type SearchConsoleSitesCmd struct {
	FailEmpty bool `name:"fail-empty" aliases:"non-empty,require-results" help:"Exit with code 3 if no results"`
}

func (c *SearchConsoleSitesCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	svc, err := newSearchConsoleService(ctx, account)
	if err != nil {
		return err
	}
	resp, err := svc.Sites.List().Context(ctx).Do()
	if err != nil {
		return err
	}

	rows := resp.SiteEntry
	if outfmt.IsJSON(ctx) {
		if err := outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"sites": rows,
		}); err != nil {
			return err
		}
		if len(rows) == 0 {
			return failEmptyExit(c.FailEmpty)
		}
		return nil
	}

	if len(rows) == 0 {
		u.Err().Println("No Search Console sites")
		return failEmptyExit(c.FailEmpty)
	}

	w, flush := tableWriter(ctx)
	defer flush()
	fmt.Fprintln(w, "SITE\tPERMISSION")
	for _, item := range rows {
		if item == nil {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\n", sanitizeTab(item.SiteUrl), sanitizeTab(item.PermissionLevel))
	}
	return nil
}

type SearchConsoleQueryCmd struct {
	SiteURL    string `arg:"" name:"siteUrl" help:"Search Console property URL (e.g. https://example.com/ or sc-domain:example.com)"`
	From       string `name:"from" required:"" help:"Start date (YYYY-MM-DD)"`
	To         string `name:"to" required:"" help:"End date (YYYY-MM-DD)"`
	Dimensions string `name:"dimensions" help:"Comma-separated dimensions (DATE,QUERY,PAGE,COUNTRY,DEVICE,SEARCH_APPEARANCE,HOUR)" default:"QUERY"`
	Type       string `name:"type" help:"Search type (WEB,IMAGE,VIDEO,NEWS,DISCOVER,GOOGLE_NEWS)" default:"WEB"`
	Max        int64  `name:"max" aliases:"limit" help:"Max rows to return (1-25000)" default:"1000"`
	Offset     int64  `name:"offset" help:"Row offset for pagination" default:"0"`
	FailEmpty  bool   `name:"fail-empty" aliases:"non-empty,require-results" help:"Exit with code 3 if no rows"`
}

func (c *SearchConsoleQueryCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	siteURL := strings.TrimSpace(c.SiteURL)
	if siteURL == "" {
		return usage("empty siteUrl")
	}

	from, err := parseSearchConsoleDate(c.From, "--from")
	if err != nil {
		return err
	}
	to, err := parseSearchConsoleDate(c.To, "--to")
	if err != nil {
		return err
	}
	if to < from {
		return usage("--to must be on or after --from")
	}

	if c.Max <= 0 || c.Max > 25000 {
		return usage("--max must be between 1 and 25000")
	}
	if c.Offset < 0 {
		return usage("--offset must be >= 0")
	}

	dimensions, err := normalizeSearchConsoleDimensions(c.Dimensions)
	if err != nil {
		return err
	}
	searchType, err := normalizeSearchConsoleType(c.Type)
	if err != nil {
		return err
	}

	svc, err := newSearchConsoleService(ctx, account)
	if err != nil {
		return err
	}
	resp, err := svc.Searchanalytics.Query(siteURL, &searchconsoleapi.SearchAnalyticsQueryRequest{
		StartDate:  from,
		EndDate:    to,
		Dimensions: dimensions,
		Type:       searchType,
		RowLimit:   c.Max,
		StartRow:   c.Offset,
	}).Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		if err := outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"site_url":                  siteURL,
			"from":                      c.From,
			"to":                        c.To,
			"type":                      searchType,
			"dimensions":                dimensions,
			"response_aggregation_type": resp.ResponseAggregationType,
			"rows":                      resp.Rows,
		}); err != nil {
			return err
		}
		if len(resp.Rows) == 0 {
			return failEmptyExit(c.FailEmpty)
		}
		return nil
	}

	if len(resp.Rows) == 0 {
		u.Err().Println("No Search Console rows")
		return failEmptyExit(c.FailEmpty)
	}

	headers := make([]string, 0, len(dimensions)+4)
	headers = append(headers, dimensions...)
	headers = append(headers, "CLICKS", "IMPRESSIONS", "CTR", "POSITION")

	w, flush := tableWriter(ctx)
	defer flush()
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range resp.Rows {
		if row == nil {
			continue
		}
		values := make([]string, 0, len(dimensions)+4)
		for i := range dimensions {
			values = append(values, sanitizeTab(searchConsoleKey(row, i)))
		}
		values = append(values,
			strconv.FormatFloat(row.Clicks, 'f', -1, 64),
			strconv.FormatFloat(row.Impressions, 'f', -1, 64),
			strconv.FormatFloat(row.Ctr, 'f', -1, 64),
			strconv.FormatFloat(row.Position, 'f', -1, 64),
		)
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}
	return nil
}

func parseSearchConsoleDate(value string, flagName string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", usagef("empty %s", flagName)
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return "", usagef("invalid %s (expected YYYY-MM-DD)", flagName)
	}
	return value, nil
}

func normalizeSearchConsoleType(raw string) (string, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if v == "" {
		return "", usage("empty --type")
	}
	switch v {
	case "WEB", "IMAGE", "VIDEO", "NEWS", "DISCOVER", "GOOGLE_NEWS":
		return v, nil
	default:
		return "", usagef("invalid --type %q (expected WEB|IMAGE|VIDEO|NEWS|DISCOVER|GOOGLE_NEWS)", raw)
	}
}

func normalizeSearchConsoleDimensions(raw string) ([]string, error) {
	parts := splitCommaList(raw)
	if len(parts) == 0 {
		return nil, nil
	}

	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.ToUpper(strings.TrimSpace(part))
		switch v {
		case "DATE", "QUERY", "PAGE", "COUNTRY", "DEVICE", "SEARCH_APPEARANCE", "HOUR":
			out = append(out, v)
		default:
			return nil, usagef("invalid dimension %q (expected DATE|QUERY|PAGE|COUNTRY|DEVICE|SEARCH_APPEARANCE|HOUR)", part)
		}
	}
	return out, nil
}

func searchConsoleKey(row *searchconsoleapi.ApiDataRow, index int) string {
	if row == nil || index < 0 || index >= len(row.Keys) {
		return ""
	}
	return row.Keys[index]
}
