package authorize

import (
	"camlistore.org/pkg/context"
	"github.com/go-errors/errors"
	. "github.com/ory-am/fosite"
	"github.com/ory-am/fosite/generator"
	"net/http"
)

// CodeResponseTypeHandler is a response handler for the Authorize Code grant using the explicit grant type
// as defined in https://tools.ietf.org/html/rfc6749#section-4.1
type CodeResponseTypeHandler struct {
	Generator generator.Generator
}

func (c *CodeResponseTypeHandler) HandleResponseType(_ context.Context, resp AuthorizeResponder, ar AuthorizeRequester, _ http.Request, session interface{}) error {
	// This let's us define multiple response types, for example open id connect's id_token
	if ar.GetResponseTypes().Has("code") {
		// Generate the code
		code, err := c.Generator.Generate()
		if err != nil {
			return errors.Wrap(err, 1)
		}

		if err := c.Store.StoreAuthorizeCodeSession(code.Signature, ar, session); err != nil {
			return errors.Wrap(err, 1)
		}

		resp.AddArgument("code", code)
		return nil
	}

	// Handler is not responsible for this request
	return ErrInvalidResponseType
}

func (c *CodeResponseTypeHandler) HandleGrantType() {

}
