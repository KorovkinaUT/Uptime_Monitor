package domain

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type Monitor struct {
	ID              uuid.UUID
	Name            string
	URL             string
	IntervalSeconds int
	TimeoutSeconds  int
	Enabled         bool
}

func (m Monitor) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidMonitor)
	}
	if utf8.RuneCountInString(m.Name) > 255 {
		return fmt.Errorf("%w: name is too long", ErrInvalidMonitor)
	}

	parsedURL, err := url.Parse(m.URL)
	if err != nil || parsedURL.Host == "" {
		return fmt.Errorf("%w: url is invalid", ErrInvalidMonitor)
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: url scheme must be http or https", ErrInvalidMonitor)
	}
	if m.IntervalSeconds <= 0 {
		return fmt.Errorf("%w: interval must be positive", ErrInvalidMonitor)
	}
	if m.TimeoutSeconds <= 0 {
		return fmt.Errorf("%w: timeout must be positive", ErrInvalidMonitor)
	}
	if m.TimeoutSeconds >= m.IntervalSeconds {
		return fmt.Errorf("%w: timeout must be less than interval", ErrInvalidMonitor)
	}

	return nil
}
