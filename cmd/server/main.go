package main

import (
	"database/sql"
	"flag"
	"log"
	"org/bredin/go-notes/pkg/index"
	"org/bredin/go-notes/pkg/routes"
	"os"

	"github.com/gofiber/fiber/v2"
)

type cliConfig struct {
	DbFileName    string
	IndexFileName string
	Port          string
}

func main() {
	config, err := parseCli(os.Args[1:])
	if err != nil {
		log.Fatal(err.Error())
	}

	db, err := sql.Open("sqlite3", config.DbFileName)
	if err != nil {
		log.Fatal(err.Error())
	}

	idx, err := index.OpenIndex(config.IndexFileName)
	if err != nil {
		log.Fatal(err.Error())
	}

	app := fiber.New()
	routes.InstallRoutes(app, db, &idx)
	log.Fatal(app.Listen(config.Port))
}

func parseCli(args []string) (cliConfig, error) {
	var config cliConfig
	fs := flag.NewFlagSet("go-notes", flag.ContinueOnError)
	fs.StringVar(&config.DbFileName, "db", "data/notes.sqlite3", "Sqlite3 backing file")
	fs.StringVar(&config.IndexFileName, "index", "data/notes.index", "Bleve index root directory")
	fs.StringVar(&config.Port, "port", ":3000", "Port serving ReST requests")
	fs.Parse(args)
	return config, nil
}
