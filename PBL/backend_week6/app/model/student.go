package model

import "time"

// Student merepresentasikan data mahasiswa di database
type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	OwnerID   int    	`json:"owner_id"` 
	CreatedAt time.Time `json:"created_at"`
}

type CreateStudentRequest struct {
	NIM  string `json:"nim"`
	Name string `json:"name"`
}

type ReplaceStudentRequest struct {
	NIM      string `json:"nim"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type PatchStudentRequest struct {
	NIM      *string `json:"nim"`
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}
