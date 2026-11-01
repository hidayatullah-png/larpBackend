package repository

import (
	"context"
	"errors"
	"fmt"

	"latihan-fiber/app/model" 

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StudentRepository interface {
	Insert(ctx context.Context, s model.Student) (model.Student, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	Delete(ctx context.Context, id int) error
	FindAll(ctx context.Context) ([]model.Student, error)
	Update(ctx context.Context, s model.Student) (model.Student, error)
	// Wajib didaftarkan di interface agar Service bisa memanggilnya
	FindAfterCursor(ctx context.Context, q model.CursorQuery) ([]model.Student, error)
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
	// Tambahkan &s.Grade di urutan yang tepat
	err := row.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.OwnerID, &s.CreatedAt)
	return s, err
}

// Kolom baku untuk query SELECT dan RETURNING (Grade ditambahkan)
const studentColumns = "id, nim, name, grade, is_active, owner_id, created_at"

func (r *studentPostgresRepository) Insert(ctx context.Context, s model.Student) (model.Student, error) {
	query := fmt.Sprintf(`
		INSERT INTO students (nim, name, grade, is_active, owner_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING %s
	`, studentColumns)

	// Default is_active selalu true saat pertama kali dibuat
	created, err := scanStudent(r.pool.QueryRow(ctx, query, s.NIM, s.Name, s.Grade, true, s.OwnerID))
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

func (r *studentPostgresRepository) FindAll(ctx context.Context) ([]model.Student, error) {
	query := fmt.Sprintf(`SELECT %s FROM students ORDER BY id DESC`, studentColumns)
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar student: %w", err)
	}
	defer rows.Close()

	var students []model.Student
	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan daftar student: %w", err)
		}
		students = append(students, s)
	}
	return students, nil
}

func (r *studentPostgresRepository) Update(ctx context.Context, s model.Student) (model.Student, error) {
	query := fmt.Sprintf(`
		UPDATE students SET nim = $1, name = $2, grade = $3, is_active = $4
		WHERE id = $5 RETURNING %s`, studentColumns)

	updated, err := scanStudent(r.pool.QueryRow(ctx, query, s.NIM, s.Name, s.Grade, s.IsActive, s.ID))
	if err != nil {
		return model.Student{}, fmt.Errorf("mengupdate student: %w", err)
	}
	return updated, nil
}

// FindAfterCursor mengambil satu halaman memakai keyset pagination
// NAMA STRUCT DIPERBAIKI: (r *studentPostgresRepository)
func (r *studentPostgresRepository) FindAfterCursor(ctx context.Context, q model.CursorQuery) ([]model.Student, error) {
	args := []any{}
	where := "WHERE 1=1"

	if q.Search != "" {
		args = append(args, "%"+q.Search+"%")
		where += fmt.Sprintf(" AND name ILIKE $%d", len(args))
	}
	if q.IsActive != nil {
		args = append(args, *q.IsActive)
		where += fmt.Sprintf(" AND is_active = $%d", len(args))
	}
	
	// Tambat posisi kursor
	if q.After != nil {
		args = append(args, q.After.CreatedAt, q.After.ID)
		// Karena urutannya DESC (terbaru ke terlama), operatornya adalah < (lebih kecil)
		where += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", len(args)-1, len(args))
	}

	// Meminta limit + 1 baris untuk mengecek hasMore
	args = append(args, q.Limit+1)
	
	// PERBAIKAN BUG 6: ORDER BY diubah menjadi DESC, DESC agar data terbaru tampil di atas
	query := fmt.Sprintf(`
		SELECT %s 
		FROM students %s 
		ORDER BY created_at DESC, id DESC 
		LIMIT $%d`, studentColumns, where, len(args)) // Memakai studentColumns agar rapi

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar mahasiswa: %w", err)
	}
	defer rows.Close()

	var result []model.Student
	for rows.Next() {
		// Gunakan helper scanStudent agar tetap konsisten
		s, err := scanStudent(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca baris mahasiswa: %w", err)
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil kueri: %w", err)
	}
	return result, nil
}