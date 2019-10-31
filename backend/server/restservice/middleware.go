package restservice

import(
	"fmt"
	"net/http"
)

// Global middleware function provides a common place to setup middlewares for
// the server.
func (r *RestAPI) GlobalMiddleware(handler http.Handler) http.Handler {
	return r.SessionManager.SessionMiddleware(handler);
};

// Checks if the user us authorized to access the system (has session).
func (r *RestAPI) Authorizer(req *http.Request) error {
	if ok, _ := r.SessionManager.Logged(req.Context()); ok {
		return nil
	}
	return fmt.Errorf("user unauthorized")
}
