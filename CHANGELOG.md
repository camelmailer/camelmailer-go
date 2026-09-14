# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-09-14

### Added

- `Campaigns`: `CreateDraft`, `CreateAndSend`, `List`, `ListForStream`,
  `Get`, `GetForStream`, `Update`, `Send`, `Cancel`. The two create methods
  hit different routes: `CreateDraft` writes the campaign and waits, while
  `CreateAndSend` expands it to the stream's subscribers before the call
  returns.
- `Subscribers`: `List`, `Add`, `Import`, `Complaint`, `Remove`.
- `Layouts`: `List`, `Create`, `Get`, `Update`, `Delete`, `UploadLogo`.
- `Inbound`: `List`, `Get`, `Retry`, `Bypass`.
- `Logs`: `List`, `Tags`.
- `Emails.SendToStream` for broadcasting to a stream's subscribers.
- `WithIdempotencyKey`, accepted by all four send methods. The key travels
  as the `Idempotency-Key` header, because the body is what the server
  hashes to recognise a replay. Passed variadically, so the existing
  signatures are unchanged.

## [0.1.0] - 2026-07-11

### Added

- Initial release of the Camelmailer Go SDK (stdlib only, zero dependencies).
- `Client` with `WithBaseURL`, `WithHTTPClient` and `WithUserAgent` options,
  plus `Client.Ping` for API-key validation.
- `Emails` service: `Send`, `SendBatch`, `SendWithTemplate`,
  `SendWithTemplateBatch`, `Get`, `List`, `Deliveries`, `Opens`, `Clicks`, `Raw`.
- `Templates` service: `List`, `Create`, `Get`, `Update`, `Archive`, `Render`.
- `Streams` service: `List`, `Create`, `Get`, `Update`, `Archive`.
- `Stats` service: `Get`, `Deliveries`.
- `Bounces` service: `List`, `Get`.
- `DMARC` service: `Summary`, `Reports`, `Report`.
- Typed API failures as `*APIError` (code, message, HTTP status),
  `errors.As`-compatible.
- Integration roundtrip test, skipped unless `CAMELMAILER_API_KEY` is set.

[Unreleased]: https://github.com/camelmailer/camelmailer-go/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/camelmailer/camelmailer-go/releases/tag/v0.2.0
[0.1.0]: https://github.com/camelmailer/camelmailer-go/releases/tag/v0.1.0
