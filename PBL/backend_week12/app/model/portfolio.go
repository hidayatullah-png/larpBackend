package model

import "time"

type Portfolio struct {
	ID           string        `json:"id"`
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type Project struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Technologies []string  `json:"technologies"`
	Link         string    `json:"link"`
	Year         int       `json:"year"`
	CreatedAt    time.Time `json:"created_at"`
}

type Certificate struct {
	Name   string `json:"name"   validate:"required,min=3,max=150"`
	Issuer string `json:"issuer" validate:"required,min=2,max=100"`
	Date   string `json:"date"   validate:"required,min=4,max=30"`
}

type UpsertPortfolioRequest struct {
	Headline     string        `json:"headline"     validate:"required,min=5,max=150"`
	Bio          string        `json:"bio"          validate:"required,min=10,max=1000"`
	Certificates []Certificate `json:"certificates" validate:"max=20,dive"`
}

type AddProjectRequest struct {
	Title        string   `json:"title"        validate:"required,min=3,max=120"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=10,dive,min=1,max=40"`
	Link         string   `json:"link"         validate:"required,url,max=255"`
	Year         int      `json:"year"         validate:"required,min=2000,max=2100"`
}

type TechStat struct {
	Technology string `json:"technology"`
	Total      int    `json:"total"`
}