package repository

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

// uuidFromPtr dereferences a *uuid.UUID, returning uuid.Nil if nil
func uuidFromPtr(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}

// uuidPtr returns a pointer to the uuid.UUID
func uuidPtr(id *uuid.UUID) *uuid.UUID {
	return id
}

// uuidToPgType converts uuid.UUID to pgtype.UUID
func uuidToPgType(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

// uuidPtrToPgType converts *uuid.UUID to pgtype.UUID
func uuidPtrToPgType(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

// pgTypeToUUID converts pgtype.UUID to *uuid.UUID
func pgTypeToUUID(pg pgtype.UUID) *uuid.UUID {
	if !pg.Valid {
		return nil
	}
	id, _ := uuid.FromBytes(pg.Bytes[:])
	return &id
}

// pgText converts string to pgtype.Text
func pgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// pgInt4 converts int to pgtype.Int4
func pgInt4(i int) pgtype.Int4 {
	if i == 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

// pgInt8 converts int64 to pgtype.Int8
func pgInt8(i int64) pgtype.Int8 {
	if i == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: i, Valid: true}
}

// pgFloat4 converts float32 to pgtype.Float4
func pgFloat4(f float32) pgtype.Float4 {
	if f == 0 {
		return pgtype.Float4{}
	}
	return pgtype.Float4{Float32: f, Valid: true}
}

// pgBool converts bool to pgtype.Bool
func pgBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}

// pgTimestamptz converts *time.Time to pgtype.Timestamptz
func pgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// pgTimestamptzToTimePtr converts pgtype.Timestamptz to *time.Time
func pgTimestamptzToTimePtr(pg pgtype.Timestamptz) *time.Time {
	if !pg.Valid {
		return nil
	}
	return &pg.Time
}

// pgInt4ToIntPtr converts pgtype.Int4 to *int
func pgInt4ToIntPtr(pg pgtype.Int4) *int {
	if !pg.Valid {
		return nil
	}
	v := int(pg.Int32)
	return &v
}

// mapToJSONB converts map[string]any to JSON bytes for JSONB column
func mapToJSONB(m map[string]any) []byte {
	if m == nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(m)
	return b
}

// jsonbToMap converts JSONB bytes to map[string]any
func jsonbToMap(b []byte) map[string]any {
	if len(b) == 0 {
		return nil
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}
