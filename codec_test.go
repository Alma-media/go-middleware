package mw

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tiny-go/codec/driver"
	"github.com/tiny-go/codec/driver/json"
	"github.com/tiny-go/codec/driver/xml"
)

func TestCodecFromList(t *testing.T) {
	type testCase struct {
		title   string
		handler http.Handler
		request *http.Request
		ctype   string
		code    int
		body    string
	}

	type Data struct {
		Test string
	}

	cases := []testCase{
		{
			title:   "should throw an error if a request codec is required but not supported",
			handler: Codec(nil, driver.DummyRegistry{&json.JSON{}, &xml.XML{}})(nil),
			request: func() *http.Request {
				r, _ := http.NewRequest(http.MethodPost, "", nil)
				r.Header.Set(contentTypeHeader, "unknown")
				r.Header.Set(contentLengthHeader, "1")
				return r
			}(),
			code: http.StatusBadRequest,
			body: "unsupported request codec: \"unknown\"\n",
		},
		{
			title:   "should ignore a request codec if not supported but not required",
			handler: Codec(nil, driver.DummyRegistry{&json.JSON{}, &xml.XML{}})(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("passed"))
				}),
			),
			request: func() *http.Request {
				r, _ := http.NewRequest(http.MethodDelete, "", nil)
				r.Header.Set(contentTypeHeader, "unknown")
				return r
			}(),
			code: http.StatusOK,
			body: "passed",
		},
		{
			title:   "should use a request codec if supported but not required",
			handler: Codec(nil, driver.DummyRegistry{&json.JSON{}, &xml.XML{}})(
				BodyClose(
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var data Data
						RequestCodecFromContext(r.Context()).Decoder(r.Body).Decode(&data)
						w.Write([]byte(data.Test))
					}),
				),
			),
			request: func() *http.Request {
				r, _ := http.NewRequest(http.MethodDelete, "", strings.NewReader("{\"test\":\"passed\"}\n"))
				r.Header.Set(contentTypeHeader, "application/json")
				r.Header.Set(contentLengthHeader, "1")
				return r
			}(),
			code: http.StatusOK,
			body: "passed",
		},
		{
			title:   "should throw an error if response codec is required but not supported",
			handler: Codec(nil, driver.DummyRegistry{&json.JSON{}, &xml.XML{}})(nil),
			request: func() *http.Request {
				r, _ := http.NewRequest(http.MethodPost, "", nil)
				r.Header.Set(contentTypeHeader, "application/json")
				r.Header.Set(contentLengthHeader, "0")
				r.Header.Set(acceptHeader, "unknown")
				return r
			}(),
			code: http.StatusBadRequest,
			body: "unsupported response codec: \"unknown\"\n",
		},
		{
			title:   "should ignore a response codec if not supported but not required",
			handler: Codec(nil, driver.DummyRegistry{&json.JSON{}, &xml.XML{}})(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("passed"))
				}),
			),
			request: func() *http.Request {
				r, _ := http.NewRequest(http.MethodDelete, "", nil)
				r.Header.Set(acceptHeader, "unknown")
				return r
			}(),
			code: http.StatusOK,
			body: "passed",
		},
		{
			title:   "should use a response codec if supported but not required",
			handler: Codec(nil, driver.DummyRegistry{&json.JSON{}, &xml.XML{}})(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					data := Data{
						Test: "passed",
					}
					ResponseCodecFromContext(r.Context()).Encoder(w).Encode(data)
				}),
			),
			request: func() *http.Request {
				r, _ := http.NewRequest(http.MethodDelete, "", nil)
				r.Header.Set(acceptHeader, "application/xml")
				return r
			}(),
			code: http.StatusOK,
			body: "<Data><Test>passed</Test></Data>",
		},
		{
			title: "should find corresponding codecs and handle the request successfully",
			handler: Codec(nil, driver.DummyRegistry{&json.JSON{}, &xml.XML{}})(
				BodyClose(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var data Data
						RequestCodecFromContext(r.Context()).Decoder(r.Body).Decode(&data)
						ResponseCodecFromContext(r.Context()).Encoder(w).Encode(data)
					}),
				),
			),
			request: func() *http.Request {
				r, _ := http.NewRequest(http.MethodPost, "", strings.NewReader("{\"test\":\"passed\"}\n"))
				r.Header.Set(contentTypeHeader, "application/json")
				r.Header.Set(contentLengthHeader, "1")
				r.Header.Set(acceptHeader, "application/xml")
				return r
			}(),
			code: http.StatusOK,
			body: "<Data><Test>passed</Test></Data>",
		},
	}

	t.Run("Given middleware function", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.title, func(t *testing.T) {
				w := httptest.NewRecorder()
				tc.handler.ServeHTTP(w, tc.request)

				if w.Code != tc.code {
					t.Errorf("status code %d was expected to be %d", w.Code, tc.code)
				}
				if w.Body.String() != tc.body {
					t.Errorf("response body %q was expected to be %q", w.Body.String(), tc.body)
				}
			})
		}
	})
}
