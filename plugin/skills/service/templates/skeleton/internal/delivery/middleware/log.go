package middleware

import (
	"github.com/arzab/gorock-catalog/rocklog"
	"github.com/arzab/gorock-kit/rockhandlers"
)

// Log logs failed requests, including decoding errors (decoding is part of
// the chain). Client errors are warnings, the rest are errors.
func Log(c rockhandlers.Context) error {
	err := c.Next()
	if err == nil {
		return nil
	}
	switch kind := rockhandlers.KindOf(err); kind {
	case rockhandlers.KindInternal, rockhandlers.KindUnavailable, rockhandlers.KindTimeout:
		rocklog.Error("request failed", rocklog.Str("route", c.Route()), rocklog.Str("kind", kind.String()), rocklog.Err(err))
	default:
		rocklog.Warn("request rejected", rocklog.Str("route", c.Route()), rocklog.Str("kind", kind.String()), rocklog.Err(err))
	}
	return err
}
