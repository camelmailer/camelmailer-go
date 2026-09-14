# camelmailer-go

[![CI](https://github.com/camelmailer/camelmailer-go/actions/workflows/ci.yml/badge.svg)](https://github.com/camelmailer/camelmailer-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/camelmailer/camelmailer-go.svg)](https://pkg.go.dev/github.com/camelmailer/camelmailer-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

The official Go SDK for [Camelmailer](https://camelmailer.com) — transactional email, nothing else. Zero dependencies, stdlib only.

## Install

```sh
go get github.com/camelmailer/camelmailer-go
```

Requires Go 1.21+.

## Quickstart

```go
client := camelmailer.NewClient("cm_xxxx")

sent, err := client.Emails.Send(ctx, &camelmailer.SendEmailRequest{
	From:     camelmailer.Address{Email: "billing@acme.com"},
	To:       []camelmailer.Address{{Email: "ada@example.com"}},
	Subject:  "Your receipt",
	TextBody: "Thanks for your purchase.",
})
```

## Self-hosted

The client defaults to the Camelmailer cloud. Point it at your own instance:

```go
client := camelmailer.NewClient("cm_xxxx",
	camelmailer.WithBaseURL("https://mail.example.com"),
	camelmailer.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
)
```

## Emails

```go
// Send with attachments, tags and metadata
sent, err := client.Emails.Send(ctx, &camelmailer.SendEmailRequest{
	From:     camelmailer.Address{Email: "billing@acme.com", Name: "Acme Billing"},
	To:       []camelmailer.Address{{Email: "ada@example.com"}},
	Subject:  "Invoice #42",
	HTMLBody: "<h1>Your invoice</h1>",
	Attachments: []camelmailer.Attachment{
		{Name: "invoice.pdf", ContentType: "application/pdf", Data: pdfBytes},
	},
	Tag:      "invoice",
	Metadata: map[string]any{"order_id": 42},
})

// Batch send — one result per entry, entries fail individually
results, err := client.Emails.SendBatch(ctx, []*camelmailer.SendEmailRequest{msg1, msg2})

// Send with a stored template
sent, err := client.Emails.SendWithTemplate(ctx, &camelmailer.SendWithTemplateRequest{
	SendEmailRequest: camelmailer.SendEmailRequest{
		From: camelmailer.Address{Email: "hello@acme.com"},
		To:   []camelmailer.Address{{Email: "ada@example.com"}},
	},
	Template:      "welcome",
	TemplateModel: map[string]any{"name": "Ada"},
})

// Retry-safe sends: the same key with the same body returns the first
// result instead of queuing a second copy, and a different body under the
// same key is refused with InvalidIdempotentRequest. All four send methods
// take it.
sent, err := client.Emails.Send(ctx, msg, camelmailer.WithIdempotencyKey("order-4711"))

// Broadcast to everyone subscribed to a stream. Recipients past the
// per-request cap of 1000 come back as Skipped, so a larger audience wants
// a campaign.
result, err := client.Emails.SendToStream(ctx, "newsletter", &camelmailer.SendToStreamRequest{
	From:     camelmailer.Address{Email: "news@acme.com"},
	Subject:  "September",
	TextBody: "What shipped this month.",
})

// Read messages
email, err := client.Emails.Get(ctx, sent.MessageID)             // message + deliveries
list, err := client.Emails.List(ctx, &camelmailer.ListEmailsOptions{Tag: "invoice"})
deliveries, err := client.Emails.Deliveries(ctx, id)
opens, err := client.Emails.Opens(ctx, id)
clicks, err := client.Emails.Clicks(ctx, id)
raw, err := client.Emails.Raw(ctx, id)                           // decoded RFC 5322 bytes
```

## Templates

```go
tmpl, err := client.Templates.Create(ctx, &camelmailer.CreateTemplateRequest{
	Name:     "Welcome",
	Subject:  "Hello {{ name }}",
	HTMLBody: "<p>Hi {{ name }}</p>",
})
templates, err := client.Templates.List(ctx)
tmpl, err = client.Templates.Get(ctx, "welcome")
tmpl, err = client.Templates.Update(ctx, "welcome", &camelmailer.UpdateTemplateRequest{
	Subject: camelmailer.String("Hi {{ name }}!"),
})
rendered, err := client.Templates.Render(ctx, "welcome", map[string]any{"name": "Ada"})
tmpl, err = client.Templates.Archive(ctx, "welcome")
```

## Streams

```go
streams, err := client.Streams.List(ctx)
stream, err := client.Streams.Create(ctx, &camelmailer.CreateStreamRequest{
	Name: "Broadcasts", StreamType: "broadcast",
})
stream, err = client.Streams.Get(ctx, "broadcasts")
stream, err = client.Streams.Update(ctx, "broadcasts", &camelmailer.UpdateStreamRequest{
	Name: camelmailer.String("Newsletter"),
})
stream, err = client.Streams.Archive(ctx, "broadcasts")
```

## Campaigns

A campaign is content plus an audience. The two ways to create one behave
differently, so pick deliberately: `CreateDraft` writes it and waits,
`CreateAndSend` expands it to the stream's subscribers before the call
returns.

```go
// Write it and leave it alone. Without ScheduledAt it stays a draft; with
// one it becomes "scheduled" and the server sends it when due.
draft, err := client.Campaigns.CreateDraft(ctx, &camelmailer.CreateDraftCampaignRequest{
	Stream:   "newsletter",
	From:     "news@acme.com",
	Name:     "September",
	Subject:  "What shipped",
	TextBody: "Hello.",
	// ScheduledAt: &sendAt,
})

// Goes out on the spot, no draft and no schedule.
sending, err := client.Campaigns.CreateAndSend(ctx, "newsletter", &camelmailer.CreateAndSendCampaignRequest{
	Name:     "Status update",
	From:     "news@acme.com",
	TextBody: "All clear.",
})

campaigns, err := client.Campaigns.List(ctx)
forStream, err := client.Campaigns.ListForStream(ctx, "newsletter")
detail, err := client.Campaigns.Get(ctx, draft.ID)       // campaign + stats
detail, err = client.Campaigns.GetForStream(ctx, "newsletter", draft.ID)

// ScheduledAt schedules; ClearSchedule drops it back to a draft. Sending
// no field at all leaves the schedule standing, so the two are separate.
_, err = client.Campaigns.Update(ctx, draft.ID, &camelmailer.UpdateCampaignRequest{ScheduledAt: &sendAt})
_, err = client.Campaigns.Update(ctx, draft.ID, &camelmailer.UpdateCampaignRequest{ClearSchedule: true})

_, err = client.Campaigns.Send(ctx, draft.ID)            // now, whatever the schedule said
_, err = client.Campaigns.Cancel(ctx, draft.ID)
```

## Subscribers

A broadcast send to an address that is not subscribed is refused, so this
list is the audience.

```go
subscribers, err := client.Subscribers.List(ctx, "newsletter")
sub, err := client.Subscribers.Add(ctx, "newsletter", &camelmailer.AddSubscriberRequest{
	Address: "ada@example.com",
	Name:    "Ada",
})
imported, err := client.Subscribers.Import(ctx, "newsletter",
	[]string{"ada@example.com", "grace@example.com"})
_, err = client.Subscribers.Complaint(ctx, "newsletter", "ada@example.com") // suppress + unsubscribe
err = client.Subscribers.Remove(ctx, "newsletter", "ada@example.com")
```

## Layouts

A layout wraps every template that uses it, so header, footer and styling
live in one place. `HTMLWrapper` has to embed the body with
`{{{ content }}}`.

```go
layouts, err := client.Layouts.List(ctx)
layout, err := client.Layouts.Create(ctx, &camelmailer.CreateLayoutRequest{
	Name:        "Default",
	Permalink:   "default",
	HTMLWrapper: "<html><body>{{{ content }}}</body></html>",
})
layout, err = client.Layouts.Get(ctx, "default")
layout, err = client.Layouts.Update(ctx, "default", &camelmailer.UpdateLayoutRequest{
	Name: camelmailer.String("Main"),
})
logoURL, err := client.Layouts.UploadLogo(ctx, "default", "data:image/png;base64,...")
err = client.Layouts.Delete(ctx, "default")
```

## Inbound and held messages

```go
held, err := client.Inbound.List(ctx, &camelmailer.ListInboundOptions{Status: "held"})
message, err := client.Inbound.Get(ctx, 55)
_, err = client.Inbound.Retry(ctx, 55)   // back on the delivery queue
_, err = client.Inbound.Bypass(ctx, 55)  // release past the hold
```

## Logs

Useful when a send did not arrive and the question is whether the request
ever reached the API.

```go
logs, err := client.Logs.List(ctx, &camelmailer.ListLogsOptions{PerPage: 25})
tags, err := client.Logs.Tags(ctx)
```

## Stats & bounces

```go
stats, err := client.Stats.Get(ctx, &camelmailer.StatsOptions{From: monthStart})
queue, err := client.Stats.Deliveries(ctx)   // outbound queue depth per domain
bounces, err := client.Bounces.List(ctx, nil)
bounce, err := client.Bounces.Get(ctx, id)
```

## DMARC

```go
summary, err := client.DMARC.Summary(ctx, &camelmailer.DMARCSummaryOptions{Domain: "acme.com"})
reports, err := client.DMARC.Reports(ctx, nil)
report, err := client.DMARC.Report(ctx, reports.Reports[0].ID)
```

## Error handling

Every API failure is a typed `*camelmailer.APIError` with the stable error code and HTTP status:

```go
sent, err := client.Emails.Send(ctx, req)
if err != nil {
	var apiErr *camelmailer.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "ValidationError", "ParameterMissing":
			// fix the request
		case "Unauthorized":
			// check the API key
		}
	}
}
```

All methods take a `context.Context` and respect cancellation and deadlines.

## Docs

Full API documentation: [camelmailer.com/docs](https://camelmailer.com/docs)

## License

[MIT](LICENSE)
