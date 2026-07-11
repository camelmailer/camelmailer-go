package camelmailer

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestDMARCSummary(t *testing.T) {
	var gotQuery map[string][]string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/dmarc/summary" {
			t.Errorf("path = %s", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		success(t, w, http.StatusOK, `{"summary":{
			"total":100,"pass":90,"fail":10,"pass_rate":0.9,
			"by_source":[{"source_ip":"203.0.113.10","count":60,"spf_aligned_pct":95.0,"dkim_aligned_pct":98.0,"disposition_counts":{"none":60}}],
			"by_disposition":{"none":95,"quarantine":5}
		}}`)
	})

	summary, err := client.DMARC.Summary(context.Background(), &DMARCSummaryOptions{
		Domain: "acme.com",
		From:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := gotQuery["domain"]; len(got) != 1 || got[0] != "acme.com" {
		t.Errorf("domain = %v", got)
	}
	if got := gotQuery["from"]; len(got) != 1 || got[0] != "2026-06-01T00:00:00Z" {
		t.Errorf("from = %v", got)
	}
	if summary.Total != 100 || summary.PassRate != 0.9 {
		t.Errorf("summary = %+v", summary)
	}
	if len(summary.BySource) != 1 || summary.BySource[0].SourceIP != "203.0.113.10" || summary.BySource[0].DispositionCounts["none"] != 60 {
		t.Errorf("by_source = %+v", summary.BySource)
	}
	if summary.ByDisposition["quarantine"] != 5 {
		t.Errorf("by_disposition = %+v", summary.ByDisposition)
	}
}

func TestDMARCReports(t *testing.T) {
	var gotQuery map[string][]string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/dmarc/reports" {
			t.Errorf("path = %s", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		success(t, w, http.StatusOK, `{
			"reports":[{
				"id":1,"domain":"acme.com","org_name":"google.com","org_email":"noreply@google.com",
				"report_id":"r-1","date_range_begin":"2026-07-01T00:00:00+00:00",
				"date_range_end":"2026-07-02T00:00:00+00:00","received_at":"2026-07-02T04:00:00+00:00",
				"record_count":3
			}],
			"pagination":{"page":1,"per_page":25,"total":1,"total_pages":1}
		}`)
	})

	reports, err := client.DMARC.Reports(context.Background(), &ListDMARCReportsOptions{
		Domain:      "acme.com",
		ListOptions: ListOptions{Page: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := gotQuery["domain"]; len(got) != 1 || got[0] != "acme.com" {
		t.Errorf("domain = %v", got)
	}
	if len(reports.Reports) != 1 || reports.Reports[0].OrgName != "google.com" || reports.Reports[0].RecordCount != 3 {
		t.Errorf("reports = %+v", reports.Reports)
	}
}

func TestDMARCReport(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/dmarc/reports/1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{
			"report":{"id":1,"domain":"acme.com","report_id":"r-1","date_range_begin":"2026-07-01T00:00:00+00:00","date_range_end":"2026-07-02T00:00:00+00:00","received_at":"2026-07-02T04:00:00+00:00","record_count":1},
			"records":[{"id":11,"source_ip":"203.0.113.10","count":4,"disposition":"none","dkim_result":"pass","spf_result":"softfail","dkim_aligned":true,"spf_aligned":false,"header_from":"acme.com","envelope_from":"bounce.acme.com"}]
		}`)
	})

	report, err := client.DMARC.Report(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if report.Report.Domain != "acme.com" {
		t.Errorf("report = %+v", report.Report)
	}
	if len(report.Records) != 1 || !report.Records[0].DKIMAligned || report.Records[0].SPFAligned {
		t.Errorf("records = %+v", report.Records)
	}
}

func TestDMARCReportNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusNotFound, "NotFound", "Resource not found")
	})
	_, err := client.DMARC.Report(context.Background(), 404)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "NotFound" {
		t.Fatalf("err = %v", err)
	}
}
