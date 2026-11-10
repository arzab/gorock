package middleware

import (
	"github.com/arzab/gorock-catalog/rocklog"
	"github.com/arzab/gorock-kit/rockhandlers"
)

// Log logs every failed request, including decoding and validation errors.
// A middleware is a handler for any chain: the general Params and Response.
func Log(c *rockhandlers.Ctx, _ rockhandlers.Params, _ rockhandlers.Response) error {
	err := c.Next()
	if err == nil {
		return nil
	}
	switch kind := rockhandlers.KindOf(err); kind {
	case rockhandlers.KindInternal, rockhandlers.KindUnavailable, rockhandlers.KindTimeout:
		rocklog.FromContext(c).Error("request failed", rocklog.Str("kind", kind.String()), rocklog.Err(err))
	default:
		rocklog.FromContext(c).Warn("request rejected", rocklog.Str("kind", kind.String()), rocklog.Err(err))
	}
	return err
}
