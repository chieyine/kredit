// Command provider-simulator supplies deterministic local implementations of
// every external connector used by Kredit. It is for development and contract
// tests only; production configuration rejects mock providers.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type simulator struct {
	mu            sync.RWMutex
	collections   map[string]map[string]any
	mandates      map[string]map[string]any
	verifications map[string]map[string]any
	messages      map[string]string
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "-healthcheck" {
		os.Exit(runSelfHealthcheck())
	}
	address := envOr("SIMULATOR_ADDR", ":8090")
	s := newSimulator()
	server := &http.Server{Addr: address, Handler: s.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	log.Printf("Kredit provider simulator listening on %s", address)
	log.Fatal(server.ListenAndServe())
}

func newSimulator() *simulator {
	return &simulator{
		collections:   make(map[string]map[string]any),
		mandates:      make(map[string]map[string]any),
		verifications: make(map[string]map[string]any),
		messages:      make(map[string]string),
	}
}

func (s *simulator) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /verifications/{kind}", s.createVerification)
	mux.HandleFunc("GET /verifications/{id}", s.getVerification)
	mux.HandleFunc("POST /mandates", s.createMandate)
	mux.HandleFunc("GET /mandates/{id}", s.getMandate)
	mux.HandleFunc("POST /mandates/{id}/cancel", s.cancelMandate)
	mux.HandleFunc("POST /mandates/{id}/restore", s.restoreMandate)
	mux.HandleFunc("POST /collections", s.createCollection)
	mux.HandleFunc("GET /collections/{id}", s.getCollection)
	mux.HandleFunc("POST /notifications", s.sendNotification)
	mux.HandleFunc("POST /documents/scan", s.scanDocument)
	return requestLimits(mux)
}

func (s *simulator) createVerification(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if kind != "person" && kind != "business" && kind != "authority" {
		writeJSON(w, 404, map[string]string{"error": "unsupported verification kind"})
		return
	}
	var input map[string]any
	if !decode(w, r, &input) {
		return
	}
	id := stableID("verification", kind, fmt.Sprint(input))
	session := map[string]any{"provider_id": id, "state": scenario(r, "verified"), "verification_level": 2, "expires_at": time.Now().UTC().Add(24 * time.Hour)}
	result := map[string]any{"provider_id": id, "subject_id": firstString(input, "subject_id", "person_id", "business_id", "representative_id"), "state": session["state"], "verification_level": 2, "safe_result": map[string]any{"matched": true, "kind": kind}}
	s.mu.Lock()
	s.verifications[id] = result
	s.mu.Unlock()
	writeJSON(w, 201, session)
}

func (s *simulator) getVerification(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	result, ok := s.verifications[r.PathValue("id")]
	if ok {
		result = clone(result)
	}
	s.mu.RUnlock()
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "verification not found"})
		return
	}
	writeJSON(w, 200, result)
}

func (s *simulator) createMandate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID        string `json:"user_id"`
		BusinessID    string `json:"business_id"`
		AmountCeiling int64  `json:"amount_ceiling"`
		Purpose       string `json:"purpose"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.UserID == "" || input.BusinessID == "" || input.AmountCeiling <= 0 {
		writeJSON(w, 422, map[string]string{"error": "user, business and positive ceiling are required"})
		return
	}
	id := stableID("mandate", input.UserID, input.BusinessID, input.Purpose)
	state := scenario(r, "active")
	mandate := map[string]any{"id": id, "provider_id": id, "user_id": input.UserID, "business_id": input.BusinessID, "status": state, "amount_ceiling_kobo": input.AmountCeiling, "amount_ceiling": input.AmountCeiling, "created_at": time.Now().UTC()}
	s.mu.Lock()
	s.mandates[id] = clone(mandate)
	s.mu.Unlock()
	writeJSON(w, 201, mandate)
}

func (s *simulator) getMandate(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	result, ok := s.mandates[r.PathValue("id")]
	if ok {
		result = clone(result)
	}
	s.mu.RUnlock()
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "mandate not found"})
		return
	}
	writeJSON(w, 200, result)
}

func (s *simulator) cancelMandate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.Reason) == "" {
		writeJSON(w, 422, map[string]string{"error": "reason is required"})
		return
	}
	s.mu.Lock()
	mandate, ok := s.mandates[r.PathValue("id")]
	if !ok {
		s.mu.Unlock()
		writeJSON(w, 404, map[string]string{"error": "mandate not found"})
		return
	}
	mandate["status"] = "cancelled"
	result := clone(mandate)
	s.mu.Unlock()
	writeJSON(w, 200, result)
}

func (s *simulator) restoreMandate(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	previous, ok := s.mandates[r.PathValue("id")]
	if !ok {
		s.mu.Unlock()
		writeJSON(w, 404, map[string]string{"error": "mandate not found"})
		return
	}
	if previous["status"] != "cancelled" && previous["status"] != "expired" && previous["status"] != "failed" {
		s.mu.Unlock()
		writeJSON(w, 409, map[string]string{"error": "mandate is not restorable"})
		return
	}
	id := stableID("mandate-restore", r.PathValue("id"), time.Now().UTC().Format(time.RFC3339Nano))
	restored := clone(previous)
	restored["id"], restored["provider_id"], restored["status"], restored["created_at"] = id, id, "active", time.Now().UTC()
	s.mandates[id] = clone(restored)
	s.mu.Unlock()
	writeJSON(w, 201, restored)
}

func (s *simulator) createCollection(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ExternalReference string `json:"external_reference"`
		AmountKobo        int64  `json:"amount_kobo"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.ExternalReference == "" || input.AmountKobo <= 0 {
		writeJSON(w, 422, map[string]string{"error": "reference and positive amount are required"})
		return
	}
	id := stableID("collection", input.ExternalReference)
	state := scenario(r, "succeeded")
	succeeded := input.AmountKobo
	if state == "partial" {
		succeeded = input.AmountKobo / 2
	}
	if state == "pending" || state == "failed" || state == "cancelled" {
		succeeded = 0
	}
	result := map[string]any{"provider_collection_id": id, "external_reference": input.ExternalReference, "state": state, "succeeded_amount_kobo": succeeded, "retryable": state == "pending" || state == "failed"}
	s.mu.Lock()
	s.collections[id] = clone(result)
	s.mu.Unlock()
	writeJSON(w, 202, result)
}

func (s *simulator) getCollection(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	result, ok := s.collections[r.PathValue("id")]
	if ok {
		result = clone(result)
	}
	s.mu.RUnlock()
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "collection not found"})
		return
	}
	if next := strings.TrimSpace(r.URL.Query().Get("state")); next != "" {
		result["state"] = next
	}
	writeJSON(w, 200, result)
}

func (s *simulator) sendNotification(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if !decode(w, r, &input) {
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeJSON(w, 422, map[string]string{"error": "Idempotency-Key is required"})
		return
	}
	s.mu.Lock()
	id, ok := s.messages[key]
	if !ok {
		id = stableID("message", key)
		s.messages[key] = id
	}
	s.mu.Unlock()
	writeJSON(w, 202, map[string]string{"message_id": id})
}

func (s *simulator) scanDocument(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if !decode(w, r, &input) {
		return
	}
	state := strings.ToUpper(scenario(r, "clean"))
	if state == "QUARANTINE" {
		state = "QUARANTINED"
	}
	if state != "CLEAN" && state != "REJECTED" && state != "QUARANTINED" {
		state = "CLEAN"
	}
	writeJSON(w, 200, map[string]string{"state": state})
}

func requestLimits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		writeDecodeError(w, err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeDecodeError(w, err)
		return false
	}
	return true
}
func writeDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func scenario(r *http.Request, fallback string) string {
	value := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Simulator-Scenario")))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scenario")))
	}
	if value == "timeout" {
		time.Sleep(1500 * time.Millisecond)
		return "pending"
	}
	if value == "" || value == "success" {
		return fallback
	}
	return value
}
func stableID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "sim_" + hex.EncodeToString(sum[:12])
}
func firstString(input map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := input[key].(string); ok && value != "" {
			return value
		}
	}
	return "simulated-subject"
}
func clone(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func runSelfHealthcheck() int {
	address := envOr("SIMULATOR_ADDR", ":8090")
	host := address
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	response, err := (&http.Client{Timeout: time.Second}).Get("http://" + host + "/healthz")
	if err != nil {
		return 1
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
