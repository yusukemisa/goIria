package main

import (
	"log"
	"os"

	"github.com/yusukemisa/goIria/iria"
)

//RFC 7233 — HTTP/1.1: Range Requests
func main() {
	downloader, err := iria.New(os.Args)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if err := downloader.Execute(); err != nil {
		log.Fatalln(err.Error())
	}
}
