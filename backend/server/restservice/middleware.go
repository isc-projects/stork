package restservice

import(
	"net/http"
	"github.com/rs/cors"
)

func (r *RestAPI) GlobalMiddleware(handler http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:4200"},
		AllowCredentials: true,
	});
	handler = c.Handler(handler);
	return r.SessionManager.SessionMiddleware(handler);
};
