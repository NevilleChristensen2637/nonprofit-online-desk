package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	presenceboard "github.com/infrai-examples/nonprofit-online-desk"
)

type presenceReader interface {
	Presence(context.Context, string) (presenceboard.Presence, error)
}

type service struct {
	client presenceReader
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	server := &http.Server{
		Addr:              ":8080",
		Handler:           (&service{client: presenceboard.NewClient(key)}).routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("nonprofit presence desk listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func (s *service) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /brief", s.brief)
	return mux
}

func (s *service) brief(w http.ResponseWriter, r *http.Request) {
	input, err := presenceboard.DecodeBriefRequest(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	presence, err := s.client.Presence(r.Context(), input.Channel)
	if err != nil {
		var apiErr *presenceboard.APIError
		if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
			writeJSON(w, apiErr.Status, map[string]string{"error": apiErr.Error(), "code": apiErr.Code})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	brief, err := presenceboard.BuildBrief(input.Channel, presence.Members, input.Assignments)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, brief)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
