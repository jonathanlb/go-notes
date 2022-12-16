package routes

import (
	"database/sql"
	"encoding/json"
	"net/url"
	"org/bredin/go-notes/pkg/auth"
	"org/bredin/go-notes/pkg/index"
	"org/bredin/go-notes/pkg/notes"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/gofiber/fiber/v2"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	jwtWare "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

func InstallRoutes(app *fiber.App, db *sql.DB, idx *bleve.Index) {
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	log = zapLogger.Sugar()

	app.Use(fiberLogger.New())
	app.Post("/login", installLogin(db))
	app.Use(jwtWare.New(jwtWare.Config{
		SigningKey: auth.GetSecret(),
	}))

	app.Get("/note/get/:noteId", installNoteGet(db))
	app.Get("/note/titles/:noteIds", installNoteGetTitles(db))
	app.Get("/note/search/:searchStr", installSearch(idx))
}

func extractNoteIdsParam(noteIdsParam string) ([]int, error) {
	noteIdsStr, err := url.QueryUnescape(noteIdsParam)
	if err != nil {
		return nil, err
	}

	noteStrIds := strings.Split(noteIdsStr, ",")
	noteIds := make([]int, len(noteStrIds))
	for i, v := range noteStrIds {
		noteIds[i], err = strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
	}
	return noteIds, nil
}

func getUserId(c *fiber.Ctx) int {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userId := int(claims["id"].(float64))
	return int(userId)
}

func installLogin(db *sql.DB) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		username := c.FormValue("user")
		password := c.FormValue("pass")
		log.Infof("Login %s", username)

		userId, err := auth.GetUserId(db, username, password)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusForbidden)
		}

		token, err := auth.GetSignedToken(username, userId)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.JSON(fiber.Map{"token": token})
	}
}

func installNoteGet(db *sql.DB) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId := getUserId(c)
		noteId, err := strconv.Atoi(c.Params("noteId"))
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}

		note, err := notes.GetNote(db, userId, noteId)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(500)
		}
		if note == nil {
			return c.SendStatus(fiber.StatusNotFound)
		}

		jsonResult, err := json.Marshal(note)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(500)
		}
		return c.SendString(string(jsonResult))
	}
}

func installNoteGetTitles(db *sql.DB) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId := getUserId(c)
		noteIds, err := extractNoteIdsParam(c.Params("noteIds"))
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}

		titles, err := notes.GetTitles(db, userId, noteIds)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		jsonResult, err := json.Marshal(titles)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendString(string(jsonResult))
	}
}

func installSearch(idx *bleve.Index) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		searchStr := c.Params("searchStr")
		searchStr, err := url.QueryUnescape(searchStr)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}

		searchHits, err := index.SearchIndex(idx, searchStr)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		jsonResult, err := json.Marshal(searchHits)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendString(string(jsonResult))
	}
}
