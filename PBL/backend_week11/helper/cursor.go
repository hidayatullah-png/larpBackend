package helper

import (
	"encoding/base64"
	"latihan-fiber/app/model"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

var ErrInvalidCursor = BadRequest("cursor tidak valid")

// EncodeCursor mengubah penanda menjadi satu string yang aman di URL
func EncodeCursor(createdAt time.Time, id int) string {
	raw := strconv.FormatInt(createdAt.UTC().UnixNano(), 10) + "|" + strconv.Itoa(id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor membaca kembali penanda dari string
func DecodeCursor(encoded string) (model.Cursor, error) {
	if strings.TrimSpace(encoded) == "" {
		return model.Cursor{}, ErrInvalidCursor
	}
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return model.Cursor{}, ErrInvalidCursor
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return model.Cursor{}, ErrInvalidCursor
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return model.Cursor{}, ErrInvalidCursor
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil || id < 1 {
		return model.Cursor{}, ErrInvalidCursor
	}
	return model.Cursor{CreatedAt: time.Unix(0, nanos).UTC(), ID: id}, nil
}

// ParseCursorQuery mengekstrak parameter pagination dari URL
func ParseCursorQuery(c *fiber.Ctx) (model.CursorQuery, error) {
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}

	q := model.CursorQuery{
		Limit:  limit,
		Search: strings.TrimSpace(c.Query("search")),
	}

	if isActiveStr := strings.TrimSpace(c.Query("is_active")); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			return q, BadRequest("is_active harus berupa true atau false")
		}
		q.IsActive = &isActive
	}

	if cursorStr := strings.TrimSpace(c.Query("cursor")); cursorStr != "" {
		cursor, err := DecodeCursor(cursorStr)
		if err != nil {
			return q, err
		}
		q.After = &cursor
	}

	return q, nil
}
