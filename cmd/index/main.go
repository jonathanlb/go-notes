package main

import (
	"flag"
	"log"
	"org/bredin/go-notes/pkg/index"
	"os"
)

type CliConfig struct {
	DbFileName    string
	IndexFileName string
}

func main() {
	config, err := parseCli(os.Args[1:])
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = index.CreateIndex(config.DbFileName, config.IndexFileName)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func parseCli(args []string) (CliConfig, error) {
	var config CliConfig
	fs := flag.NewFlagSet("index-notes", flag.ContinueOnError)
	fs.StringVar(&config.DbFileName, "db", "data/notes.sqlite3", "Sqlite3 backing file")
	fs.StringVar(&config.IndexFileName, "index", "data/notes.index", "Bleve index root directory")
	fs.Parse(args)
	return config, nil
}
