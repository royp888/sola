package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type uuidCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func encodeUUIDCursor(createdAt time.Time, id uuid.UUID) string {
	payload, err := json.Marshal(uuidCursor{CreatedAt: createdAt.UTC(), ID: id.String()})
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(payload)
}

func decodeUUIDCursor(cursor string) (time.Time, uuid.UUID, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return time.Time{}, uuid.UUID{}, fmt.Errorf("invalid cursor")
	}
	var payload uuidCursor
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return time.Time{}, uuid.UUID{}, fmt.Errorf("invalid cursor")
	}
	parsedID, err := uuid.Parse(strings.TrimSpace(payload.ID))
	if err != nil || payload.CreatedAt.IsZero() {
		return time.Time{}, uuid.UUID{}, fmt.Errorf("invalid cursor")
	}
	return payload.CreatedAt, parsedID, nil
}
