package main

import (
	"log"
	"net/http"
	"simulator_service"
)

func main() {
	cfg := simulator_service.ConfigFromEnv()

	kafkaWriter := simulator_service.NewWriter(cfg)
	if err := simulator_service.CreateTopicIfNotExists(cfg); err != nil {
		log.Fatal(err)
	}

	mux := simulator_service.NewMux(kafkaWriter, []byte("some-key"))

	if err := http.ListenAndServe(":"+cfg.HttpPort, mux); err != nil {
		log.Fatal(err)
	}
}
