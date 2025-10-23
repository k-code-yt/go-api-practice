package main

import (
	"flag"
	"os"

	"poseur.com/dotenv"
)

var (
	ENV string = ""
)

func loadENV() {
	var envfile = flag.String("env", ".env", "environment file")
	flag.Parse()
	_ = dotenv.SetenvFile(*envfile)
	ENV = os.Getenv("ENV")
}

func main() {
	loadENV()
	initProm()
	// chatClientLoop()
	chatServer()
}
