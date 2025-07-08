package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/schubergphilis/rpi_exporter/pkg/export/prometheus"
	"github.com/schubergphilis/rpi_exporter/pkg/mbox"
	log "github.com/sirupsen/logrus"
)

var (
	flagAddr  = flag.String("addr", "", "Listen on address")
	flagDebug = flag.Bool("debug", false, "Print debug messages")
)

func main() {
	flag.Parse()

	mbox.Debug = *flagDebug

	if *flagAddr != "" {
		http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := prometheus.Write(w); err != nil {
				log.Printf("Error: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}))
		log.Printf("Listening on %s", *flagAddr)
		http.ListenAndServe(*flagAddr, nil)

		return
	}

	if err := prometheus.Write(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
