package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Text converts a string to a valid pgtype.Text.
// Use this when building params structs that contain nullable VARCHAR fields.
func Text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// NullText returns an invalid (NULL) pgtype.Text.
func NullText() pgtype.Text {
	return pgtype.Text{}
}

func BigInt(i int64) pgtype.Int8 {
	return pgtype.Int8{Int64: i, Valid: true}
}

func UUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func NullUUID() pgtype.UUID {
	return pgtype.UUID{}
}

// NullableUUID wraps a uuid.UUID, returning NULL for the zero UUID value.
func NullableUUID(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return NullUUID()
	}
	return UUID(id)
}

// StringFromNull returns the string value of a nullable text field, or "" if null.
func StringFromNull(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

// UUIDFromPg extracts a uuid.UUID from a pgtype.UUID, returning zero UUID if invalid.
func UUIDFromPg(u pgtype.UUID) uuid.UUID {
	if !u.Valid {
		return uuid.UUID{}
	}
	return u.Bytes
}

// NullString returns an invalid (NULL) pgtype.Text.
func NullString() pgtype.Text {
	return pgtype.Text{}
}

// JSONB serializes a map into a JSONB byte slice.
// Returns {} if the input is nil.
func JSONB(data map[string]any) []byte {
	if data == nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(data)
	return b
}

// FromJSONB deserializes a JSONB byte slice into a map.
// Returns nil if the input is empty.
func FromJSONB(data []byte) map[string]any {
	if len(data) == 0 {
		return nil
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	return m
}

// GetCommitByHashStr looks up a commit by its hash string.
func (q *Queries) GetCommitByHashStr(ctx context.Context, hash string) (Commit, error) {
	return q.GetCommitByHash(ctx, Text(hash))
}
