package settings_test

import (
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// testHTTPServer wraps httptest.Server, matching testutil's *httptest.Server
// so testutil.Get/Post/Do helpers can be reused. Unlike testutil.NewServer,
// this variant does not auto-mount auth.Middleware — the setup function may
// register public and authenticated routes side by side.
type testHTTPServer struct {
	raw *httptest.Server
}

func (s *testHTTPServer) Close() { s.raw.Close() }

func newTestHTTPServer(t *testing.T, mount func(r chi.Router)) *testHTTPServer {
	t.Helper()
	r := chi.NewRouter()
	mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &testHTTPServer{raw: srv}
}
