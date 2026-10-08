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
