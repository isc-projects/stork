package restservice

import(
	"net/http"
)

// Global middleware function provides a common place to setup middlewares for
// the server.
func (r *RestAPI) GlobalMiddleware(handler http.Handler) http.Handler {
	return r.SessionManager.SessionMiddleware(handler);
};
