package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type EncryptPayload struct {
	Message     string        `json:"message" redis:"message"`
	AccessCount int           `json:"access_count" redis:"access_count"`
	Expiry      time.Duration `json:"expiry"`
}

func (enc *EncryptPayload) Validate() error {
	// Max expiry shouldn't exceed 7 days
	if enc.Expiry > 86400*7 {
		return fmt.Errorf("expiry exceeds the max allowed limit")
	}
	if enc.AccessCount > 30 {
		return fmt.Errorf("access_count exceeds the max allowed limit")
	}
	return nil
}

type EncryptPayloadOut struct {
	UUID string `json:"uuid"`
}

// wrap is a middleware that wraps HTTP handlers and injects the "app" context.
func wrap(app *App, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "app", app)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// resp is used to send uniform response structure.
type resp struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// sendResponse sends a JSON envelope to the HTTP response.
func sendResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	out, err := json.Marshal(resp{Status: "success", Data: data})
	if err != nil {
		sendErrorResponse(w, "Internal Server Error.", http.StatusInternalServerError, nil)
		return
	}

	w.Write(out)
}

// sendErrorResponse sends a JSON error envelope to the HTTP response.
func sendErrorResponse(w http.ResponseWriter, message string, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	resp := resp{
		Status:  "error",
		Message: message,
		Data:    data,
	}
	out, err := json.Marshal(resp)
	if err != nil {
		sendErrorResponse(w, "Internal Server Error.", http.StatusInternalServerError, nil)
		return
	}

	w.Write(out)
}

// Handler for encrypting payload.
func handleEncrypt(w http.ResponseWriter, r *http.Request) {
	var (
		app = r.Context().Value("app").(*App)
	)

	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		lo.Printf("error reading request body: %v", err)
		sendErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest, nil)
		return
	}
	var payload EncryptPayload
	if err := json.Unmarshal(b, &payload); err != nil {
		lo.Printf("error unmarshalling payload: %v\n", err)
		sendErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest, nil)
		return
	}

	if err := payload.Validate(); err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// Set a default expiry.
	if payload.Expiry == 0 {
		payload.Expiry = time.Hour * 24
	} else {
		payload.Expiry = time.Duration(payload.Expiry) * time.Second
	}

	// Set a default access count
	if payload.AccessCount == 0 {
		payload.AccessCount = 1
	}

	// Generate a UUID and store the encrypted message in Redis.
	uuid := uuid.New()
	err = app.storePayload(uuid.String(), payload)
	if err != nil {
		lo.Printf("error storing payload in redis: %v\n", err)
		sendErrorResponse(w, "Error storing payload", http.StatusInternalServerError, nil)
		return
	}

	// Return the UUID in reponse.
	resp := EncryptPayloadOut{
		UUID: uuid.String(),
	}
	sendResponse(w, http.StatusOK, resp)
}

// Handler for looking up encrypted payload.
func handleLookup(w http.ResponseWriter, r *http.Request) {
	var (
		app  = r.Context().Value("app").(*App)
		uuid = chi.URLParam(r, "uuid")
	)

	// Lookup for the key.
	data, err := app.fetchPayload(uuid)
	if err != nil {
		lo.Printf("error fetching payload: %v\n", err)
		sendErrorResponse(w, "Error fetching message", http.StatusInternalServerError, nil)
		return
	}
	// Check the access count.
	if data.AccessCount < 0 {
		sendErrorResponse(w, "Max attempts reached", http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, http.StatusOK, data)
}

func (app *App) storePayload(uuid string, payload EncryptPayload) error {
	ctx := context.Background()

	if err := app.redis.HSet(ctx, uuid, "message", payload.Message).Err(); err != nil {
		return err
	}
	if err := app.redis.HIncrBy(ctx, uuid, "access_count", int64(payload.AccessCount)).Err(); err != nil {
		return err
	}
	if err := app.redis.Expire(ctx, uuid, payload.Expiry).Err(); err != nil {
		return err
	}

	return nil
}

func (app *App) fetchPayload(uuid string) (EncryptPayload, error) {
	ctx := context.Background()
	var out EncryptPayload

	// Check if key exists.
	if !app.redis.HExists(ctx, uuid, "message").Val() {
		return out, fmt.Errorf("no message stored for uuid: %s", uuid)
	}

	// Decrement the access count.
	if err := app.redis.HIncrBy(ctx, uuid, "access_count", int64(-1)).Err(); err != nil {
		return out, err
	}

	// Scan the keys in the struct.
	if err := app.redis.HGetAll(ctx, uuid).Scan(&out); err != nil {
		return out, err
	}

	// Get the TTL
	out.Expiry = app.redis.TTL(ctx, uuid).Val() / time.Second

	// Remove the key if max access has reached.
	if out.AccessCount <= 0 {
		if err := app.redis.Del(ctx, uuid).Err(); err != nil {
			return out, err
		}
	}

	return out, nil
}
