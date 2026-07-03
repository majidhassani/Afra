// Package validator provides small request-validation helpers used by
// application services (never by HTTP handlers directly).
package validator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	apperrors "casemind/pkg/errors"
)

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Problems accumulates field validation failures.
type Problems struct {
	fields []string
}

func New() *Problems { return &Problems{} }

func (p *Problems) Required(field, value string) *Problems {
	if strings.TrimSpace(value) == "" {
		p.fields = append(p.fields, field+" is required")
	}
	return p
}

func (p *Problems) Email(field, value string) *Problems {
	if value != "" && !emailRe.MatchString(value) {
		p.fields = append(p.fields, field+" must be a valid email")
	}
	return p
}

func (p *Problems) MinLen(field, value string, n int) *Problems {
	if utf8.RuneCountInString(value) < n {
		p.fields = append(p.fields, fmt.Sprintf("%s must be at least %d characters", field, n))
	}
	return p
}

func (p *Problems) MaxLen(field, value string, n int) *Problems {
	if utf8.RuneCountInString(value) > n {
		p.fields = append(p.fields, fmt.Sprintf("%s must be at most %d characters", field, n))
	}
	return p
}

func (p *Problems) OneOf(field, value string, allowed ...string) *Problems {
	if value == "" {
		return p
	}
	for _, a := range allowed {
		if value == a {
			return p
		}
	}
	p.fields = append(p.fields, fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", ")))
	return p
}

// Err returns an invalid-kind error listing all problems, or nil.
func (p *Problems) Err() error {
	if len(p.fields) == 0 {
		return nil
	}
	return apperrors.Invalid("validation_failed", strings.Join(p.fields, "; "))
}
