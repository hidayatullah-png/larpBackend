package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"latihan-fiber/app/model"
)

var (
	ErrSupervisorFull    = errors.New("kuota bimbingan dosen sudah penuh")
	ErrAlreadySupervised = errors.New("mahasiswa sudah memiliki dosen pembimbing")
	ErrNotSupervisedBy   = errors.New("mahasiswa tidak sedang dibimbing oleh dosen asal tersebut")
)

type SupervisorRepository interface {
	WithTx(tx DBTX) SupervisorRepository
	FindAll(ctx context.Context) ([]model.Supervisor, error)
	FindByID(ctx context.Context, id int) (model.Supervisor, error)
	LockStudentEntity(ctx context.Context, studentID int) error
	LockSupervisors(ctx context.Context, ids ...int) error
	IncrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error)
	DecrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error)
	InsertSupervision(ctx context.Context, supervisorID, studentID int) error
	DeleteSupervision(ctx context.Context, supervisorID, studentID int) error
	UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error)
}

const supervisorColumns = `id, nidn, name, quota, supervised_count, version, created_at`

type supervisorPostgresRepository struct{ db DBTX }

func NewSupervisorRepository(db DBTX) SupervisorRepository {
	return &supervisorPostgresRepository{db: db}
}

func (r *supervisorPostgresRepository) WithTx(tx DBTX) SupervisorRepository {
	return &supervisorPostgresRepository{db: tx}
}

func scanSupervisor(row pgx.Row) (model.Supervisor, error) {
	var s model.Supervisor
	err := row.Scan(&s.ID, &s.NIDN, &s.Name, &s.Quota, &s.SupervisedCount, &s.Version, &s.CreatedAt)
	return s, err
}

func (r *supervisorPostgresRepository) FindAll(ctx context.Context) ([]model.Supervisor, error) {
	rows, err := r.db.Query(ctx, "SELECT "+supervisorColumns+" FROM supervisors ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar pembimbing: %w", err)
	}
	defer rows.Close()

	result := []model.Supervisor{}
	for rows.Next() {
		s, err := scanSupervisor(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca baris pembimbing: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *supervisorPostgresRepository) FindByID(ctx context.Context, id int) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(ctx,
		"SELECT "+supervisorColumns+" FROM supervisors WHERE id = $1", id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrNotFound
		}
		return model.Supervisor{}, fmt.Errorf("mengambil pembimbing: %w", err)
	}
	return s, nil
}

// LockStudentEntity mengunci baris mahasiswa di tabel students dengan FOR UPDATE
func (r *supervisorPostgresRepository) LockStudentEntity(ctx context.Context, studentID int) error {
	var id int
	err := r.db.QueryRow(ctx,
		"SELECT id FROM students WHERE id = $1 FOR UPDATE", studentID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("mengunci data student: %w", err)
	}
	return nil
}

// LockSupervisors mengunci kedua dosen pembimbing secara berurutan (ORDER BY id) untuk mencegah deadlock
func (r *supervisorPostgresRepository) LockSupervisors(ctx context.Context, ids ...int) error {
	rows, err := r.db.Query(ctx,
		"SELECT id FROM supervisors WHERE id = ANY($1) ORDER BY id FOR UPDATE", ids)
	if err != nil {
		return fmt.Errorf("mengunci dosen pembimbing: %w", err)
	}
	defer rows.Close()

	found := 0
	for rows.Next() {
		found++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("mengunci dosen pembimbing: %w", err)
	}
	if found != len(ids) {
		return ErrNotFound
	}
	return nil
}

func (r *supervisorPostgresRepository) IncrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(ctx, `
		UPDATE supervisors SET supervised_count = supervised_count + 1
		WHERE id = $1 AND supervised_count < quota
		RETURNING `+supervisorColumns, supervisorID))
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, fmt.Errorf("menambah jumlah bimbingan: %w", err)
	}
	if _, findErr := r.FindByID(ctx, supervisorID); findErr != nil {
		return model.Supervisor{}, findErr
	}
	return model.Supervisor{}, ErrSupervisorFull
}

func (r *supervisorPostgresRepository) DecrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(ctx, `
		UPDATE supervisors SET supervised_count = supervised_count - 1
		WHERE id = $1
		RETURNING `+supervisorColumns, supervisorID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrNotFound
		}
		return model.Supervisor{}, fmt.Errorf("mengurangi jumlah bimbingan: %w", err)
	}
	return s, nil
}

func (r *supervisorPostgresRepository) InsertSupervision(ctx context.Context, supervisorID, studentID int) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO supervisions (supervisor_id, student_id) VALUES ($1, $2)",
		supervisorID, studentID)
	if err != nil {
		if pgErrorCode(err) == codeUniqueViolation {
			return ErrAlreadySupervised
		}
		return fmt.Errorf("mencatat bimbingan: %w", err)
	}
	return nil
}

func (r *supervisorPostgresRepository) DeleteSupervision(ctx context.Context, supervisorID, studentID int) error {
	tag, err := r.db.Exec(ctx,
		"DELETE FROM supervisions WHERE supervisor_id = $1 AND student_id = $2",
		supervisorID, studentID)
	if err != nil {
		return fmt.Errorf("menghapus bimbingan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotSupervisedBy
	}
	return nil
}

func (r *supervisorPostgresRepository) UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error) {
	s, err := scanSupervisor(r.db.QueryRow(ctx, `
		UPDATE supervisors SET quota = $1, version = version + 1
		WHERE id = $2 AND version = $3 AND $1 >= supervised_count
		RETURNING `+supervisorColumns, quota, id, version))
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, fmt.Errorf("mengubah kuota pembimbing: %w", err)
	}

	current, findErr := r.FindByID(ctx, id)
	if findErr != nil {
		return model.Supervisor{}, findErr
	}
	if current.Version != version {
		return model.Supervisor{}, ErrVersionConflict
	}
	return model.Supervisor{}, ErrQuotaBelowFilled
}