package camelmailer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DMARCService reads stored DMARC aggregate reports and compliance
// summaries via /api/v2/server/dmarc.
type DMARCService struct {
	client *Client
}

// DMARCSource is one sending source in a DMARC summary (top 20 by
// volume).
type DMARCSource struct {
	// SourceIP is the sending IP address.
	SourceIP string `json:"source_ip"`
	// Count is the number of covered messages from this source.
	Count int64 `json:"count"`
	// SPFAlignedPct is the percentage of SPF-aligned messages.
	SPFAlignedPct float64 `json:"spf_aligned_pct"`
	// DKIMAlignedPct is the percentage of DKIM-aligned messages.
	DKIMAlignedPct float64 `json:"dkim_aligned_pct"`
	// DispositionCounts counts messages per disposition.
	DispositionCounts map[string]int64 `json:"disposition_counts"`
}

// DMARCSummary aggregates the stored DMARC aggregate-report rows.
type DMARCSummary struct {
	// Total is the number of messages covered by the reports.
	Total int64 `json:"total"`
	// Pass counts messages with DKIM and SPF aligned.
	Pass int64 `json:"pass"`
	// Fail counts messages failing alignment.
	Fail int64 `json:"fail"`
	// PassRate is Pass / Total (0.0–1.0).
	PassRate float64 `json:"pass_rate"`
	// BySource lists the top 20 sending sources by volume.
	BySource []DMARCSource `json:"by_source"`
	// ByDisposition counts messages per disposition.
	ByDisposition map[string]int64 `json:"by_disposition"`
}

// DMARCReport is one stored DMARC aggregate report.
type DMARCReport struct {
	// ID is the numeric report id.
	ID int64 `json:"id"`
	// Domain is the reported domain.
	Domain string `json:"domain"`
	// OrgName is the reporting organization, e.g. "google.com".
	OrgName string `json:"org_name"`
	// OrgEmail is the reporting organization's contact address.
	OrgEmail string `json:"org_email"`
	// ReportID is the reporter's report id.
	ReportID string `json:"report_id"`
	// DateRangeBegin is the start of the covered range.
	DateRangeBegin time.Time `json:"date_range_begin"`
	// DateRangeEnd is the end of the covered range.
	DateRangeEnd time.Time `json:"date_range_end"`
	// ReceivedAt is when the report was ingested.
	ReceivedAt time.Time `json:"received_at"`
	// RecordCount is the number of rows in the report.
	RecordCount int64 `json:"record_count"`
}

// DMARCRecord is one row of a DMARC aggregate report.
type DMARCRecord struct {
	// ID is the numeric record id.
	ID int64 `json:"id"`
	// SourceIP is the sending IP address.
	SourceIP string `json:"source_ip"`
	// Count is the number of messages this row covers.
	Count int64 `json:"count"`
	// Disposition is "none", "quarantine" or "reject".
	Disposition string `json:"disposition"`
	// DKIMResult is the raw DKIM result, e.g. "pass".
	DKIMResult string `json:"dkim_result"`
	// SPFResult is the raw SPF result, e.g. "softfail".
	SPFResult string `json:"spf_result"`
	// DKIMAligned reports DKIM identifier alignment.
	DKIMAligned bool `json:"dkim_aligned"`
	// SPFAligned reports SPF identifier alignment.
	SPFAligned bool `json:"spf_aligned"`
	// HeaderFrom is the From-header domain.
	HeaderFrom string `json:"header_from"`
	// EnvelopeFrom is the envelope-sender domain.
	EnvelopeFrom string `json:"envelope_from"`
}

// DMARCReportDetail is one report with its rows, as returned by
// DMARCService.Report.
type DMARCReportDetail struct {
	// Report is the aggregate report.
	Report DMARCReport `json:"report"`
	// Records lists the report's rows.
	Records []DMARCRecord `json:"records"`
}

// DMARCSummaryOptions filters DMARCService.Summary. Zero values are
// omitted; From/To match reports whose date range overlaps the window.
type DMARCSummaryOptions struct {
	// Domain restricts to one reported domain.
	Domain string
	// From is the inclusive window start.
	From time.Time
	// To is the inclusive window end.
	To time.Time
}

// ListDMARCReportsOptions filters DMARCService.Reports.
type ListDMARCReportsOptions struct {
	ListOptions
	// Domain restricts to one reported domain.
	Domain string
	// From is the inclusive window start.
	From time.Time
	// To is the inclusive window end.
	To time.Time
}

// ListDMARCReportsResult is one page of DMARC aggregate reports.
type ListDMARCReportsResult struct {
	// Reports is the page of reports, newest report range first.
	Reports []DMARCReport `json:"reports"`
	// Pagination describes the page window.
	Pagination Pagination `json:"pagination"`
}

// Summary returns the DMARC compliance summary over the stored
// aggregate-report rows. opts may be nil.
//
// API: GET /api/v2/server/dmarc/summary
func (s *DMARCService) Summary(ctx context.Context, opts *DMARCSummaryOptions) (*DMARCSummary, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Domain != "" {
			query.Set("domain", opts.Domain)
		}
		setTime(query, "from", opts.From)
		setTime(query, "to", opts.To)
	}
	var out struct {
		Summary DMARCSummary `json:"summary"`
	}
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/dmarc/summary", query, nil, &out); err != nil {
		return nil, err
	}
	return &out.Summary, nil
}

// Reports returns the stored DMARC aggregate reports, filtered and
// paginated. opts may be nil.
//
// API: GET /api/v2/server/dmarc/reports
func (s *DMARCService) Reports(ctx context.Context, opts *ListDMARCReportsOptions) (*ListDMARCReportsResult, error) {
	query := url.Values{}
	if opts != nil {
		opts.ListOptions.values(query)
		if opts.Domain != "" {
			query.Set("domain", opts.Domain)
		}
		setTime(query, "from", opts.From)
		setTime(query, "to", opts.To)
	}
	out := new(ListDMARCReportsResult)
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/dmarc/reports", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Report returns one stored DMARC report with its records.
//
// API: GET /api/v2/server/dmarc/reports/{id}
func (s *DMARCService) Report(ctx context.Context, id int64) (*DMARCReportDetail, error) {
	out := new(DMARCReportDetail)
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/server/dmarc/reports/%d", id), nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
