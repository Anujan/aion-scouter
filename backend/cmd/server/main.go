// Command server runs the Aion Scouter API (and, optionally, serves the built
// frontend).
//
//	go run ./cmd/server -addr :8080 -static ../frontend/build
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/anujan/aion-scouter/backend/internal/api"
	"github.com/anujan/aion-scouter/backend/internal/gamedata"
	"github.com/anujan/aion-scouter/backend/internal/ncsoft"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	static := flag.String("static", "", "directory of the built frontend to serve (optional)")
	flag.Parse()

	data, err := gamedata.Load("global")
	if err != nil {
		log.Fatalf("load game data: %v", err)
	}
	srv := &api.Server{Data: data, Source: &api.Source{Live: ncsoft.NewClient()}, StaticDir: *static}
	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      90 * time.Second,
	}
	log.Printf("aion-scouter API listening on %s (%d classes loaded)", *addr, len(data.Classes))
	log.Fatal(httpSrv.ListenAndServe())
}
