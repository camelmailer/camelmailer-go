package camelmailer

import "net/http"

// SendOption tunes one send call. Go has no optional arguments, so the
// send methods take these variadically: the common call stays two
// arguments and nothing about the existing signatures changes.
type SendOption func(*sendConfig)

// sendConfig collects the applied SendOptions.
type sendConfig struct {
	idempotencyKey string
}

// WithIdempotencyKey makes a send replayable. Sending the same key with
// the same body returns the original result instead of queuing a second
// copy; the same key with a different body is refused with
// InvalidIdempotentRequest (HTTP 409) rather than silently ignored.
//
// Keys are scoped to the server and a completed result is kept for 24
// hours. All four send endpoints honour them.
func WithIdempotencyKey(key string) SendOption {
	return func(cfg *sendConfig) {
		cfg.idempotencyKey = key
	}
}

// header builds the request header for the applied options, or nil when
// there is nothing to send.
func (cfg sendConfig) header() http.Header {
	if cfg.idempotencyKey == "" {
		return nil
	}
	// A header, not a body field: the body is what the server hashes to
	// recognise the same request.
	return http.Header{"Idempotency-Key": []string{cfg.idempotencyKey}}
}

// applySendOptions folds the options into one config.
func applySendOptions(opts []SendOption) sendConfig {
	var cfg sendConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}
