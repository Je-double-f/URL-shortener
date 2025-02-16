package save

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	resp "url-shortener/internal/lib/api/response"
	generatingalias "url-shortener/internal/lib/generating_alias"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"golang.org/x/exp/slog"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

//go:generate go run github.com/vektra/mockery/v2@v2.28.2 --name=URLSaver
type URLSaver interface {
	SaveURL(urlToSave string, alias string) (int64, error)
}

func New(log *slog.Logger, urlSaver URLSaver, srv *http.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if errors.Is(err, io.EOF) {
			// Такую ошибку встретим, если получили запрос с пустым телом.
			// Обработаем её отдельно
			log.Error("request body is empty")

			render.JSON(w, r, resp.Error("empty request"))

			return
		}
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.JSON(w, r, resp.Error("failed to decode request"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))

			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		alias := req.Alias
		aliasLength := 1
		pointerOne, pointerTwo, pointerThree, pointerFour := -1, -1, -1, -1

		if alias == "" {
			// Используем switch для выбора нужной функции в зависимости от aliasLength
			switch aliasLength {
			case 1:
				alias = generatingalias.NewGeneratedAliasOneSize(&aliasLength, &pointerOne)
			case 2:
				alias = generatingalias.NewGeneratedAliasTwoSize(&aliasLength, &pointerOne, &pointerTwo)
			case 3:
				alias = generatingalias.NewGeneratedAliasThreeSize(&aliasLength, &pointerOne, &pointerTwo, &pointerThree)
			case 4:
				alias = generatingalias.NewGeneratedAliasFourSize(&pointerOne, &pointerTwo, &pointerThree, &pointerFour)
			default:
				log.Info("Stopping server due to invalid or too long aliasLength")

				// Настроим тайм-аут для корректной остановки сервера
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				if err := srv.Shutdown(ctx); err != nil {
					log.Error("Failed to stop server", sl.Err(err))
					return
				}

				log.Info("Server stopped successfully")
				return // Закрытие функции, после того как сервер был остановлен
			}
		}

		id, err := urlSaver.SaveURL(req.URL, alias)
		if errors.Is(err, storage.ErrURLExists) {
			log.Info("url already exists", slog.String("url", req.URL))

			render.JSON(w, r, resp.Error("url already exists"))

			return
		}
		if err != nil {
			log.Error("failed to add url", sl.Err(err))

			render.JSON(w, r, resp.Error("failed to add url"))

			return
		}

		log.Info("url added", slog.Int64("id", id))

		responseOK(w, r, alias)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}
