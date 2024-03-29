package routes

import (
	"encoding/json"
	"net/url"
	"org/bredin/go-notes/pkg/auth"
	"org/bredin/go-notes/pkg/index"
	"org/bredin/go-notes/pkg/notes"
	"strconv"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	jwtWare "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

func InstallRoutes(app *fiber.App, dbFileName string, idx *bleve.Index) {
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	log = zapLogger.Sugar()

	app.Use(fiberLogger.New())
	app.Use(cors.New(cors.Config{
		AllowMethods:  "GET, POST",
		AllowOrigins:  "*",
		AllowHeaders:  "Accept, Authorization, Content-Type, Origin, user, pass",
		ExposeHeaders: "Accept, Authorization, Content-Type, Origin, user, pass",
	}))

	// Static is installed before jwt checks because client
	// MD viewer will have difficultly injecting
	// auth info into request.
	app.Static("/public", "./data/public")

	app.Post("/login", installLogin(dbFileName))
	app.Use(jwtWare.New(jwtWare.Config{
		SigningKey: auth.GetSecret(),
	}))

	app.Post("/note/create", installNoteCreate(dbFileName, idx))
	app.Get("/note/privacy/:noteId/:privacy", installUpdateNotePrivacy(dbFileName))
	app.Get("/note/get/:noteId", installNoteGet(dbFileName))
	app.Get("/note/recent/:numNotes", installRecent(dbFileName))
	app.Get("/note/search/:searchStr", installSearch(idx))
	app.Get("/user/get/:userId", installUserGet(dbFileName))
}

func getUserId(c *fiber.Ctx) int {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userId := int(claims["id"].(float64))
	return int(userId)
}

func installLogin(dbFileName string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		username := c.FormValue("user")
		password := c.FormValue("pass")
		log.Infof("Login %s", username)

		db, err := notes.OpenNoteDb(dbFileName)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		defer db.Close()

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
		return c.JSON(fiber.Map{"token": token, "id": userId})
	}
}

func installNoteCreate(dbFileName string, idx *bleve.Index) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId := getUserId(c)
		content, err := url.QueryUnescape(
			c.FormValue("content"))
		if err != nil {
			log.Errorf("Create cannot unescape query")
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}
		note := notes.NoteRecord{
			Author:     userId,
			Content:    content,
			Created:    int(time.Now().Unix()),
			Privacy:    notes.DEFAULT_ACCESS,
			RenderHint: 1,
		}

		db, err := notes.OpenNoteDb(dbFileName)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		defer db.Close()

		id, err := notes.CreateNote(db, &note)
		if err != nil {
			msg := err.Error()
			log.Errorf("Save: %s", msg)
			c.SendString(msg)
			return c.SendStatus(500)
		}
		res := c.SendString(strconv.Itoa(id))
		// TODO: do in background
		err = (*idx).Index(strconv.Itoa(id), note)
		if err != nil {
			log.Errorf("Cannot update index: %s", err.Error())
		}
		return res
	}
}

func installNoteGet(dbFileName string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId := getUserId(c)
		noteId, err := strconv.Atoi(c.Params("noteId"))
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}

		db, err := notes.OpenNoteDb(dbFileName)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		defer db.Close()

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
func installRecent(dbFileName string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId := getUserId(c)
		numNotes, err := strconv.Atoi(c.Params("numNotes"))
		if err != nil || numNotes <= 0 {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}

		db, err := notes.OpenNoteDb(dbFileName)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		defer db.Close()

		searchHits, err := notes.GetRecentNotes(db, userId, numNotes)
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

func installSearch(idx *bleve.Index) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		searchStr := c.Params("searchStr")
		searchStr, err := url.QueryUnescape(searchStr)
		if err != nil {
			log.Errorf("Search cannot unescape query")
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}
		log.Infof("Search %s", searchStr)

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

func installUpdateNotePrivacy(dbFileName string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId := getUserId(c)
		noteId, err := strconv.Atoi(c.Params("noteId"))
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}

		privacy, err := strconv.Atoi(c.Params("privacy"))
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}

		db, err := notes.OpenNoteDb(dbFileName)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		defer db.Close()

		err = notes.SetNotePrivacy(db, userId, noteId, privacy)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(500)
		}
		return c.SendString("OK")
	}
}

func installUserGet(dbFileName string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId, err := strconv.Atoi(c.Params("userId"))
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusBadRequest)
		}

		db, err := notes.OpenNoteDb(dbFileName)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		defer db.Close()

		author, err := notes.GetAuthor(db, userId)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(500)
		}
		if author == nil {
			return c.SendStatus(fiber.StatusNotFound)
		}

		jsonResult, err := json.Marshal(author)
		if err != nil {
			c.SendString(err.Error())
			return c.SendStatus(500)
		}
		return c.SendString(string(jsonResult))
	}
}
