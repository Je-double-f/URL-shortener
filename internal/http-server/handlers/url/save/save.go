package save

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"url-shortener/internal/config"
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

// @Summary      Создать сокращенный URL
// @Description  Принимает длинный URL и создает для него короткую версию
// @Accept       json
// @Produce      json
// @Param        request body Request true "URL для сокращения"
// @Success      200 {object} Response
// @Failure      400 {object} Response
// @Failure      500 {object} Response
// @Router       / [post]
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
			log.Error("request body is empty")
			render.JSON(w, r, resp.Error("empty request"))
			return
		}
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
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

		if req.Alias != "" {
			log.Error("Don't allowed to set alias manually", slog.String("alias", req.Alias))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("manual alias setting is not allowed"))
			return
		}

		cfg := config.MustLoad()
		db, err := sql.Open("sqlite3", cfg.StoragePath)
		if err != nil {
			log.Error("failed to open database", sl.Err(err))
			render.JSON(w, r, resp.Error("internal server error"))
			return
		}
		defer db.Close()

		var alias string
		var aliasLength, pointerOne, pointerTwo, pointerThree, pointerFour int

		_ = db.QueryRow("SELECT value FROM alias_value WHERE name = 'AliasLength'").Scan(&aliasLength)
		_ = db.QueryRow("SELECT value FROM alias_value WHERE name = 'PointerOne'").Scan(&pointerOne)
		_ = db.QueryRow("SELECT value FROM alias_value WHERE name = 'PointerTwo'").Scan(&pointerTwo)
		_ = db.QueryRow("SELECT value FROM alias_value WHERE name = 'PointerThree'").Scan(&pointerThree)
		_ = db.QueryRow("SELECT value FROM alias_value WHERE name = 'PointerFour'").Scan(&pointerFour)

		switch aliasLength {
		case 1:
			alias = generatingalias.NewGeneratedAliasOneSize(&aliasLength, &pointerOne)
		case 2:
			alias = generatingalias.NewGeneratedAliasTwoSize(&aliasLength, &pointerOne, &pointerTwo)
		case 3:
			alias = generatingalias.NewGeneratedAliasThreeSize(&aliasLength, &pointerOne, &pointerTwo, &pointerThree)
		case 4:
			alias = generatingalias.NewGeneratedAliasFourSize(&aliasLength, &pointerOne, &pointerTwo, &pointerThree, &pointerFour)
		default:
			log.Info("No free aliases left. Stopping server.")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				log.Error("Failed to stop server", sl.Err(err))
			} else {
				log.Info("Server stopped successfully")
			}
			return
		}

		// Сохраняем новые значения в БД
		res, err := db.Exec("UPDATE alias_value SET value = ? WHERE name = 'AliasLength'", aliasLength)
		if err != nil {
			log.Error("failed to update AliasLength", sl.Err(err))
			return
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			log.Warn("AliasLength not updated, possibly missing record")
		}

		res, err = db.Exec("UPDATE alias_value SET value = ? WHERE name = 'PointerOne'", pointerOne)
		if err != nil {
			log.Error("failed to update PointerOne", sl.Err(err))
			return
		}
		rowsAffected, _ = res.RowsAffected()
		if rowsAffected == 0 {
			log.Warn("PointerOne not updated, possibly missing record")
		}

		res, err = db.Exec("UPDATE alias_value SET value = ? WHERE name = 'PointerTwo'", pointerTwo)
		if err != nil {
			log.Error("failed to update PointerTwo", sl.Err(err))
			return
		}
		rowsAffected, _ = res.RowsAffected()
		if rowsAffected == 0 {
			log.Warn("PointerTwo not updated, possibly missing record")
		}

		res, err = db.Exec("UPDATE alias_value SET value = ? WHERE name = 'PointerThree'", pointerThree)
		if err != nil {
			log.Error("failed to update PointerThree", sl.Err(err))
			return
		}
		rowsAffected, _ = res.RowsAffected()
		if rowsAffected == 0 {
			log.Warn("PointerThree not updated, possibly missing record")
		}

		res, err = db.Exec("UPDATE alias_value SET value = ? WHERE name = 'PointerFour'", pointerFour)
		if err != nil {
			log.Error("failed to update AliasLength", sl.Err(err))
			return
		}
		rowsAffected, _ = res.RowsAffected()
		if rowsAffected == 0 {
			log.Warn("PointerFour not updated, possibly missing record")
		}

		id, err := urlSaver.SaveURL(req.URL, alias)
		if errors.Is(err, storage.ErrURLExists) {
			log.Info("url already exists", slog.String("url", req.URL))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error(fmt.Sprintf("url with alias: %s already exists", alias)))
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
