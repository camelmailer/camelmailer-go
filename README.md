# camelmailer-go

[![CI](https://github.com/camelmailer/camelmailer-go/actions/workflows/ci.yml/badge.svg)](https://github.com/camelmailer/camelmailer-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/camelmailer/camelmailer-go.svg)](https://pkg.go.dev/github.com/camelmailer/camelmailer-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

The official Go SDK for [CamelMailer](https://camelmailer.com) — transactional email, nothing else. Zero dependencies, stdlib only.

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

The client defaults to the CamelMailer cloud. Point it at your own instance:

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
