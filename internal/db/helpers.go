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

// StringFromNull returns the string value of a nullable text field, or "" if null.
func StringFromNull(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

// GetCommitByHash looks up a commit by its hash string.
func (q *Queries) GetCommitByHashStr(ctx context.Context, hash string) (Commit, error) {
	return q.GetCommitByHash(ctx, Text(hash))
}
