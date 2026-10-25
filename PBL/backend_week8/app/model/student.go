package model

import "time"

// Student merepresentasikan data mahasiswa di database
type Student struct {
    ID        int       `json:"id"`
    NIM       string    `json:"nim"`
    Name      string    `json:"name"`
    Grade     float64   `json:"grade"` 
    IsActive  bool      `json:"is_active"`
    OwnerID   int       `json:"owner_id"` 
    CreatedAt time.Time `json:"created_at"`
}

type CreateStudentRequest struct {
    NIM   string  `json:"nim" validate:"required,numeric,len=9"`
    Name  string  `json:"name" validate:"required,min=3,max=100"`
    Grade float64 `json:"grade" validate:"min=0,max=100"`
}

type ReplaceStudentRequest struct {
    NIM      string  `json:"nim" validate:"required,numeric,len=9"`
    Name     string  `json:"name" validate:"required,min=3,max=100"`
    Grade    float64 `json:"grade" validate:"min=0,max=100"`
    IsActive bool    `json:"is_active"`
}

type PatchStudentRequest struct {
    NIM      *string  `json:"nim,omitempty" validate:"omitnil,numeric,len=9"`
    Name     *string  `json:"name,omitempty" validate:"omitnil,min=3,max=100"`
    Grade    *float64 `json:"grade,omitempty" validate:"omitnil,min=0,max=100"`
    IsActive *bool    `json:"is_active,omitempty"`
}
type Cursor struct {
	CreatedAt time.Time
	ID        int
}

type CursorQuery struct {
	Limit    int
	Search   string
	IsActive *bool
	After    *Cursor
}

// CursorMeta menggantikan Meta lama pada endpoint yang memakai kursor[cite: 41].
type CursorMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}