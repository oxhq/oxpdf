package oxpdf

type openConfig struct {
	password string
	strict   bool
}

// OpenOption configures PDF opening.
type OpenOption func(*openConfig)

// WithPassword requests password-based opening for encrypted PDFs.
func WithPassword(password string) OpenOption {
	return func(cfg *openConfig) {
		cfg.password = password
	}
}

// WithStrictParsing asks the backing engine to reject malformed PDFs more eagerly.
func WithStrictParsing() OpenOption {
	return func(cfg *openConfig) {
		cfg.strict = true
	}
}
