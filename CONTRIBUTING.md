# Contributing

Thanks for helping improve camelmailer-go!

## Dev setup

Go 1.21+ is the only requirement — the SDK has zero dependencies.

```sh
git clone https://github.com/camelmailer/camelmailer-go
cd camelmailer-go
go test ./... -race
```

## Checks

CI runs these on every push/PR; run them locally first:

```sh
gofmt -l .          # must print nothing
go vet ./...
go test ./... -race
```

## Conventions

- Stdlib only — no third-party dependencies.
- Test-first: every resource method and error path has a unit test
  against `httptest.Server` (no network in unit tests).
- Every exported symbol carries a doc comment.
- Response shapes follow the CamelMailer OpenAPI spec; the envelope
  (`status`/`time`/`data`|`error`) is handled centrally in `Client.do`.

## Integration test

`TestIntegrationRoundtrip` runs against a real instance when
`CAMELMAILER_API_KEY` (and optionally `CAMELMAILER_BASE_URL`,
`CAMELMAILER_TEST_FROM`, `CAMELMAILER_TEST_TO`) is set. It is skipped in CI.
