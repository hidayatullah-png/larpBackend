package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"latihan-fiber/app/model"
)

// Sentinel error: error milik lapisan repository, bukan error milik pgx[cite: 1].
var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
)

// StudentRepository adalah KONTRAK penyimpanan data mahasiswa[cite: 1].
// Tidak ada kata "SQL" atau "postgres" di sini[cite: 1].
type StudentRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	FindByNIM(ctx context.Context, nim string) (model.Student, error)
	Create(ctx context.Context, s model.Student) (model.Student, error)
	Update(ctx context.Context, s model.Student) (model.Student, error)
	Delete(ctx context.Context, id int) error
}

// kolomUrut adalah daftar putih: pemetaan dari nilai yang boleh dikirim klien
// ke nama kolom yang sebenarnya[cite: 1]. ORDER BY tidak dapat memakai parameter,
// sehingga daftar putih inilah yang mencegah SQL injection[cite: 1].
var kolomUrut = map[string]string{
	"id":          "id",
	"nim":         "nim",
	"name":        "name",
	"grade":       "grade",
	"mata_kuliah": "mata_kuliah",
	"created_at":  "created_at",
}

type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

// NewStudentRepository mengembalikan interface, bukan struct konkret[cite: 1].
func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

// buildFilter menyusun bagian WHERE beserta argumennya[cite: 1].
// Nilai dari klien SELALU menjadi argumen ($1, $2, ...), tidak pernah disambung langsung[cite: 1].
func buildFilter(q model.ListQuery) (string, []any) {
	where := " WHERE 1=1"
	args := []any{}

	if q.Search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d)", len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}

	if q.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
		args = append(args, *q.IsActive)
	}

	return where, args
}

func (r *studentPostgresRepository) FindAll(ctx context.Context, q model.ListQuery) ([]model.Student, int, error) {
	where, args := buildFilter(q)

	// 1) Hitung total sebelum dipenggal, untuk keperluan meta[cite: 1].
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM students"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung student: %w", err)
	}

	// 2) Ambil satu halaman saja[cite: 1].
	arah := "ASC"
	if q.Order == "desc" {
		arah = "DESC"
	}

	sqlText := fmt.Sprintf(
		`SELECT s.id, s.nim, s.name, COALESCE (n.grade, 0),COALESCE (n.mata_kuliah, ''), s.is_active, s.created_at 
		FROM students s
		LEFT JOIN nilai n ON s.id = n.id_student %s 
		ORDER BY %s %s 
		LIMIT $%d OFFSET $%d`,
		where, kolomUrut[q.Sort], arah, len(args)+1, len(args)+2,
	)

	args = append(args, q.Limit, q.Offset())

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar student: %w", err)
	}
	defer rows.Close() // Wajib ditutup agar koneksi kembali ke pool[cite: 1].

	hasil := []model.Student{}
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.MataKuliah, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("membaca baris student: %w", err)
		}
		hasil = append(hasil, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}

	return hasil, total, nil
}

func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT s.id, s.nim, s.name, COALESCE (n.grade, 0),COALESCE (n.mata_kuliah, ''), s.is_active, s.created_at 
		FROM students s
		LEFT JOIN nilai n ON s.id = n.id_student 
		WHERE s.id = $1`, id,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.MataKuliah, &s.IsActive, &s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound // pgx.ErrNoRows diterjemahkan menjadi error milik kita[cite: 1].
		}
		return model.Student{}, fmt.Errorf("mengambil student: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) FindByNIM(ctx context.Context, nim string) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		`SELECT s.id, s.nim, s.name, COALESCE (n.grade, 0),COALESCE (n.mata_kuliah, ''), s.is_active, s.created_at 
		FROM students s
		LEFT JOIN nilai n ON s.id = n.id_student  
		WHERE s.nim =$1`, nim,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.MataKuliah, &s.IsActive, &s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil student berdasarkan nim: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) Create(ctx context.Context, s model.Student) (model.Student, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Student{}, fmt.Errorf("Memulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	// RETURNING membuat id dan created_at hasil buatan basis data langsung ikut kembali[cite: 1].
	err = tx.QueryRow(ctx,
		`INSERT INTO students (nim, name, is_active) 
		VALUES ($1, $2, $3) 
		RETURNING id, created_at`,
		s.NIM, s.Name, s.IsActive,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("menyimpan student: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO nilai (id_student, grade, mata_kuliah)
		VALUES ($1, $2, $3)`,
		s.ID, s.Grade, s.MataKuliah,
	)

	if err != nil {
		return model.Student{}, fmt.Errorf("Menyimpan nilai student: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Student{}, fmt.Errorf("Commit transaksi create: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) Update(ctx context.Context, s model.Student) (model.Student, error) {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Student{}, fmt.Errorf("Memulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx,
		`UPDATE students SET nim = $1, name = $2, is_active = $3 
		WHERE id = $4
		RETURNING true`,
		s.NIM, s.Name, s.IsActive, s.ID,
	).Scan(&exists)

	if err != nil || !exists {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("Memperbarui student: %w", err)
	}
	_, err = tx.Exec(ctx,
		`UPDATE nilai SET mata_kuliah = $1, grade = $2
		WHERE id_student =$3`,
		s.MataKuliah, s.Grade, s.ID,
	)
	if err != nil {
		return model.Student{}, fmt.Errorf("Memperbarui nilai student: %w", err)
	}
	err = tx.QueryRow(ctx, `SELECT created_at FROM students WHERE id = $1`, s.ID).Scan(&s.CreatedAt)
	if err != nil {
		return model.Student{}, fmt.Errorf("mengambil waktu pembuatan: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Student{}, fmt.Errorf("commit transaksi update: %w", err)
	}
	return s, nil
}

func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM students WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus student: %w", err)
	}

	// Perintah berhasil dijalankan, tetapi tidak ada baris yang terkena. Artinya id tidak ada[cite: 1].
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// isUniqueViolation memeriksa apakah error berasal dari pelanggaran batasan UNIQUE.
// Kode 23505 adalah kode resmi PostgreSQL untuk itu[cite: 1].
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
