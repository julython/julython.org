package db

import (
	"context"

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

// GetCommitByHashStr looks up a commit by its hash string.
func (q *Queries) GetCommitByHashStr(ctx context.Context, hash string) (Commit, error) {
	return q.GetCommitByHash(ctx, Text(hash))
}
