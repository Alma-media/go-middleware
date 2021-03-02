package mw

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/tiny-go/codec"
	"github.com/tiny-go/errors"
)

const (
	acceptHeader            = "Accept"
	contentTypeHeader       = "Content-Type"
	contentLengthHeader     = "Content-Length"
	transferEncodingHeader  = "Transfer-Encoding"
	defaultTransferEncoding = "identity"
)

// codecKey is a private unique key that is used to put/get codec from the context.
type codecKey struct{ kind string }

// Codecs represents any kind of codec registry, thus it can be global registry
// or a custom list of codecs that is supposed to be used for particular route.
type Codecs interface {
	// Lookup should find appropriate Codec by MimeType or return nil if not found.
	Lookup(mimeType string) codec.Codec
}

// Codec middleware searches for suitable request/response codecs according to
// "Content-Type"/"Accept" headers and puts the correct codecs into the context.
func Codec(fn errors.HandlerFunc, codecs Codecs) Middleware {
	if fn == nil {
		fn = http.Error
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var reqCodec, resCodec codec.Codec
			// get request codec
			if reqCodec = codecs.Lookup(r.Header.Get(contentTypeHeader)); reqCodec == nil {
				if isContentTypeHeaderRequired(r) {
					fn(w, fmt.Sprintf("unsupported request codec: %q", r.Header.Get(contentTypeHeader)), http.StatusBadRequest)
					return
				}
			} else {
				r = r.WithContext(context.WithValue(r.Context(), codecKey{"req"}, reqCodec))
			}
			// get response codec
			if resCodec = codecs.Lookup(r.Header.Get(acceptHeader)); resCodec == nil {
				if isAcceptHeaderRequired(r) {
					fn(w, fmt.Sprintf("unsupported response codec: %q", r.Header.Get(acceptHeader)), http.StatusBadRequest)
					return
				}
			} else {
				r = r.WithContext(context.WithValue(r.Context(), codecKey{"res"}, resCodec))
			}
			// call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RequestCodecFromContext pulls the Codec from a request context or returns nil.
func RequestCodecFromContext(ctx context.Context) codec.Codec {
	codec, _ := ctx.Value(codecKey{"req"}).(codec.Codec)
	return codec
}

// ResponseCodecFromContext pulls the Codec from a request context or returns nil.
func ResponseCodecFromContext(ctx context.Context) codec.Codec {
	codec, _ := ctx.Value(codecKey{"res"}).(codec.Codec)
	return codec
}

// isContentTypeHeaderRequired returns the HTTP method request body type requirement.
// By RFC7231 (https://tools.ietf.org/html/rfc7231) only POST, PUT and PATCH methods
// should contain a request body. DELETE method body is optional.
func isContentTypeHeaderRequired(r *http.Request) bool {
	switch r.Method {
	// Body is required
	case http.MethodPost: fallthrough
	case http.MethodPut: fallthrough
	case http.MethodPatch:
		return shouldRequestBodyBeProcessed(r, true)
	// May have body, but not required
	case http.MethodDelete:
		return shouldRequestBodyBeProcessed(r, false)
	// No body
	case http.MethodGet: fallthrough
	case http.MethodHead: fallthrough
	case http.MethodConnect: fallthrough
	case http.MethodOptions: fallthrough
	case http.MethodTrace: fallthrough
	default:
		return false
	}
}

// isAcceptHeaderRequired returns the HTTP method response body type requirement.
// By RFC7231 (https://tools.ietf.org/html/rfc7231) only GET, POST, CONNECT,
// OPTIONS and PATCH methods should indicate the details of a response body.
// DELETE method response body is optional.
func isAcceptHeaderRequired(r *http.Request) bool {
	switch r.Method {
	// Body is required
	case http.MethodGet: fallthrough
	case http.MethodPost: fallthrough
	case http.MethodConnect: fallthrough
	case http.MethodOptions: fallthrough
	case http.MethodPatch:
		return true
	// May have body, but not required
	case http.MethodDelete: fallthrough
	// No body
	case http.MethodHead: fallthrough
	case http.MethodPut: fallthrough
	case http.MethodTrace: fallthrough
	default:
		return false
	}
}

func shouldRequestBodyBeProcessed(r *http.Request, required bool) bool {
	transferEncoding := r.Header.Get(transferEncodingHeader)
	hasRequestBody := transferEncoding != "" && !strings.EqualFold(transferEncoding, defaultTransferEncoding)

	hasRequestBody = hasRequestBody || func() bool {
		contentLengthStr := r.Header.Get(contentLengthHeader)
		if contentLengthStr != "" {
			contentLength, err := strconv.Atoi(contentLengthStr)
			if err != nil || contentLength < 0 {
				return false
			}

			return contentLength > 0
		}

		return required
	}()

	return hasRequestBody
}
