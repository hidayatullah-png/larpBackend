package repository

import (
	"context"
	"errors"
	"fmt"

	"latihan-fiber/app/model" // Sesuaikan dengan nama module go.mod milikmu

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


type StudentRepository interface {
	Insert(ctx context.Context, s model.Student) (model.Student, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	Delete(ctx context.Context, id int) error
}

type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

// Helper untuk membaca baris dari database agar tidak berulang
func scanStudent(row pgx.Row) (model.Student, error) {
	var s model.Student
	err := row.Scan(&s.ID, &s.NIM, &s.Name, &s.IsActive, &s.OwnerID, &s.CreatedAt)
	return s, err
}

// Kolom baku untuk query SELECT dan RETURNING
const studentColumns = "id, nim, name, is_active, owner_id, created_at"

func (r *studentPostgresRepository) Insert(ctx context.Context, s model.Student) (model.Student, error) {
	query := fmt.Sprintf(`
		INSERT INTO students (nim, name, is_active, owner_id)
		VALUES ($1, $2, $3, $4)
		RETURNING %s
	`, studentColumns)

	// Default is_active selalu true saat pertama kali dibuat
	created, err := scanStudent(r.pool.QueryRow(ctx, query, s.NIM, s.Name, true, s.OwnerID))
	if err != nil {
		return model.Student{}, fmt.Errorf("menyimpan student: %w", err)
	}
	return created, nil
}

func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (model.Student, error) {
	query := fmt.Sprintf(`SELECT %s FROM students WHERE id = $1`, studentColumns)

	s, err := scanStudent(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mencari student: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM students WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("menghapus student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
