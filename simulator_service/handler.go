package simulator_service

import (
	"github.com/segmentio/kafka-go"
	"io"
	"net/http"
)

func NewCreateOrderHandler(writer *kafka.Writer, key []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		err = writer.WriteMessages(r.Context(),
			kafka.Message{
				Key:   key,
				Value: body,
			},
		)

		if err != nil {
			http.Error(w, "Error sending message to Kafka: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func NewMux(writer *kafka.Writer, key []byte) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", NewCreateOrderHandler(writer, key))
	return mux
}
