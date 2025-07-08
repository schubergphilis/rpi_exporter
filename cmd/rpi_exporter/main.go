package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/schubergphilis/rpi_exporter/pkg/export/prometheus"
	"github.com/schubergphilis/rpi_exporter/pkg/mbox"
	log "github.com/sirupsen/logrus"
)

var (
	flagAddr  = flag.String("addr", "", "Listen on address")
	flagDebug = flag.Bool("debug", false, "Print debug messages")
)

const (
	httpReadTimeout  = 5 * time.Second
	httpWriteTimeout = 10 * time.Second
	httpIdleTimeout  = 120 * time.Second
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

		srv := &http.Server{
			Addr:         *flagAddr,
			Handler:      nil,
			ReadTimeout:  httpReadTimeout,
			WriteTimeout: httpWriteTimeout,
			IdleTimeout:  httpIdleTimeout,
		}
		if err := srv.ListenAndServe(); err != nil {
			log.WithError(err).Fatal("unable to listen and serve http")
		}

		return
	}

	if err := prometheus.Write(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
