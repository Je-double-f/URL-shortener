package save

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	resp "url-shortener/internal/lib/api/response"
	generatingalias "url-shortener/internal/lib/generating_alias"
	"url-shortener/internal/lib/logger/sl"

	// "url-shortener/internal/lib/random"
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

const aliasFile = "Alias.txt"          // Файл для хранения aliasLength
const firstFile = "Firstpointer.txt"   // Файл для хранения pointerOne
const secondFile = "Secondpointer.txt" // Файл для хранения pointerTwo
const thirdFile = "Thirdpointer.txt"   // Файл для хранения pointerThree
const fourthFile = "Fourthpointer.txt" // Файл для хранения pointerFour

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

		// Проверяем, передан ли alias вручную
		if req.Alias != "" {
			log.Error("Don't allowed to set alias manually", slog.String("alias", req.Alias))

			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("manual alias setting is not allowed"))

			return
		}

		alias := ""
		aliasLength := loadAliasLength()

		if alias == "" {
			pointerOne := loadPointerOne()
			pointerTwo := loadPointerTwo()
			pointerThree := loadPointerThree()
			pointerFour := loadPointerFour()
			switch aliasLength {
			case 1:
				alias = generatingalias.NewGeneratedAliasOneSize(&aliasLength, &pointerOne)
				savePointerOne(pointerOne)
				if alias == "full" {
					alias = generatingalias.NewGeneratedAliasTwoSize(&aliasLength, &pointerOne, &pointerTwo)
					log.Info("Alias with size 1 is full")
				}
			case 2:
				alias = generatingalias.NewGeneratedAliasTwoSize(&aliasLength, &pointerOne, &pointerTwo)
				savePointerTwo(pointerTwo)
				if alias == "full" {
					alias = generatingalias.NewGeneratedAliasThreeSize(&aliasLength, &pointerOne, &pointerTwo, &pointerThree)
					log.Info("Alias with size 2 is full")
				}
			case 3:
				alias = generatingalias.NewGeneratedAliasThreeSize(&aliasLength, &pointerOne, &pointerTwo, &pointerThree)
				savePointerThree(pointerThree)
				if alias == "full" {
					alias = generatingalias.NewGeneratedAliasFourSize(&aliasLength, &pointerOne, &pointerTwo, &pointerThree, &pointerFour)
					log.Info("Alias with size 3 is full")
				}
			case 4:
				alias = generatingalias.NewGeneratedAliasFourSize(&aliasLength, &pointerOne, &pointerTwo, &pointerThree, &pointerFour)
				savePointerFour(pointerFour)
				if alias == "full" {
					log.Info("Alias with size 4 if full")
				}
			default:
				log.Info("Stopping server due to invalid or you have no free alias")

				// Настроим тайм-аут для корректной остановки сервера
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				log.Info(fmt.Sprintf("aliasLength: %d", aliasLength))
				if err := srv.Shutdown(ctx); err != nil {
					log.Error("Failed to stop server", sl.Err(err))
					return
				}

				log.Info("Server stopped successfully")
				return // Закрытие функции, после того как сервер был остановлен
			}

			saveAliasLegth(aliasLength)
		}

		id, err := urlSaver.SaveURL(req.URL, alias)
		if errors.Is(err, storage.ErrURLExists) {
			log.Info("url already exists", slog.String("url", req.URL))

			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, resp.Error("url with alias: {"+alias+"}, already exists"))

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

func loadAliasLength() int {
	data, err := os.ReadFile(aliasFile)
	if err != nil {
		return 1
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return value
}

func loadPointerOne() int {
	data, err := os.ReadFile(firstFile)
	if err != nil {
		return 0
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return value
}

func loadPointerTwo() int {
	data, err := os.ReadFile(secondFile)
	if err != nil {
		return 0
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return value
}

func loadPointerThree() int {
	data, err := os.ReadFile(thirdFile)
	if err != nil {
		return 0
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return value
}

func loadPointerFour() int {
	data, err := os.ReadFile(fourthFile)
	if err != nil {
		return 0
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return value
}

func saveAliasLegth(pointer int) {
	os.WriteFile(aliasFile, []byte(strconv.Itoa(pointer)), 0644)
}

func savePointerOne(pointer int) {
	os.WriteFile(firstFile, []byte(strconv.Itoa(pointer)), 0644)
}
func savePointerTwo(pointer int) {
	os.WriteFile(secondFile, []byte(strconv.Itoa(pointer)), 0644)
}
func savePointerThree(pointer int) {
	os.WriteFile(thirdFile, []byte(strconv.Itoa(pointer)), 0644)
}
func savePointerFour(pointer int) {
	os.WriteFile(fourthFile, []byte(strconv.Itoa(pointer)), 0644)
}
