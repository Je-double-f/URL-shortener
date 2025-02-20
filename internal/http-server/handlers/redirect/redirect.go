package redirect

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"golang.org/x/exp/slog"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage"
)

// URLGetter is an interface for getting url by alias.
//
//go:generate go run github.com/vektra/mockery/v2@v2.28.2 --name=URLGetter
type URLGetter interface {
	GetURL(alias string) (string, error)
}

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// @Summary Redirect to original URL
// @Description Перенаправляет пользователя на оригинальный URL по его короткому идентификатору
// @Param alias path string true "Short URL alias"
// @Success 200 "Successfully redirected"
// @Success 302 "Moved Temporarily"
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /{alias} [get]
func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.redirect.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("alias is empty")

			render.JSON(w, r, resp.Error("invalid request"))

			return
		}

		resURL, err := urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", "alias", alias)

			render.JSON(w, r, resp.Error("not found"))

			return
		}
		if err != nil {
			log.Error("failed to get url", sl.Err(err))

			render.JSON(w, r, resp.Error("internal error"))

			return
		}

		log.Info("got url", slog.String("url", resURL))

		// redirect to found url
		http.Redirect(w, r, resURL, http.StatusFound)
	}
}
