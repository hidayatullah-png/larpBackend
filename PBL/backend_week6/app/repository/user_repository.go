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

// Sentinel error: error milik lapisan repository, bukan error milik pgx.
// Lapisan atas cukup mengenal dua ini dan tidak perlu tahu basis datanya apa.
var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
)

// UserRepository adalah KONTRAK penyimpanan data user.
type UserRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.User, int, error)
	FindByID(ctx context.Context, id int) (model.User, error)
	FindByUsername(ctx context.Context, username string) (model.User, error) // DITAMBAHKAN KHUSUS UNTUK AUTHENTICATION
	Create(ctx context.Context, u model.User) (model.User, error)
	Update(ctx context.Context, u model.User) (model.User, error)
	Delete(ctx context.Context, id int) error
}

// kolomUrut adalah daftar putih: pemetaan dari nilai yang boleh dikirim klien
// ke nama kolom yang sebenarnya.
var kolomUrut = map[string]string{
	"id":         "id",
	"username":   "username",
	"email":      "email",
	"role":       "role",
	"created_at": "created_at",
}

// kolomUser adalah SATU-SATUNYA daftar kolom yang dibaca dari tabel users.
// Urutannya HARUS sama dengan urutan Scan di scanUser. Dengan satu sumber,
// kolom tidak akan lagi terlewat di salah satu query (penyebab role kosong).
const kolomUser = `id, username, email, password, role, is_active, created_at`

// scanUser membaca satu baris ke model.User. pgx.Row dan pgx.Rows sama-sama
// memenuhi interface ini, jadi helper ini bisa dipakai di QueryRow maupun rows.Next().
func scanUser(row pgx.Row, u *model.User) error {
	return row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)
}

type userPostgresRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository mengembalikan interface, bukan struct konkret.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userPostgresRepository{pool: pool}
}

// buildFilter menyusun bagian WHERE beserta argumennya.
func buildFilter(q model.ListQuery) (string, []any) {
	where := " WHERE 1 = 1"
	args := []any{}

	if q.Search != "" {
		where += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)", len(args)+1, len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}

	if q.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", len(args)+1)
		args = append(args, *q.IsActive)
	}

	return where, args
}

func (r *userPostgresRepository) FindAll(ctx context.Context, q model.ListQuery) ([]model.User, int, error) {
	where, args := buildFilter(q)

	// 1) Hitung total sebelum dipenggal, untuk keperluan meta.
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung user: %w", err)
	}

	// 2) Ambil satu halaman saja.
	arah := "ASC"
	if q.Order == "desc" {
		arah = "DESC"
	}

	sqlText := fmt.Sprintf(
		`SELECT %s FROM users%s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		kolomUser, where, kolomUrut[q.Sort], arah, len(args)+1, len(args)+2,
	)

	args = append(args, q.Limit, q.Offset())
	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar user: %w", err)
	}
	defer rows.Close()

	hasil := []model.User{}
	for rows.Next() {
		var u model.User
		if err := scanUser(rows, &u); err != nil {
			return nil, 0, fmt.Errorf("membaca baris user: %w", err)
		}
		hasil = append(hasil, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}

	return hasil, total, nil
}

func (r *userPostgresRepository) FindByID(ctx context.Context, id int) (model.User, error) {
	var u model.User
	err := scanUser(
		r.pool.QueryRow(ctx, `SELECT `+kolomUser+` FROM users WHERE id = $1`, id),
		&u,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}
	return u, nil
}

// FindByUsername ditambahkan agar AuthService bisa mencari password DAN role saat proses Login.
func (r *userPostgresRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var u model.User
	err := scanUser(
		r.pool.QueryRow(ctx, `SELECT `+kolomUser+` FROM users WHERE username = $1`, username),
		&u,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengambil user berdasarkan username: %w", err)
	}
	return u, nil
}

// Create menyimpan role dari u.Role. Pastikan service (register) mengisi role
// default dan JANGAN mengambilnya dari body request klien.
func (r *userPostgresRepository) Create(ctx context.Context, u model.User) (model.User, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role, is_active) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		u.Username, u.Email, u.Password, u.Role, u.IsActive,
	).Scan(&u.ID, &u.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.User{}, ErrDuplicate
		}
		return model.User{}, fmt.Errorf("menyimpan user: %w", err)
	}
	return u, nil
}

// Update sengaja TIDAK mengubah role. Perubahan role sebaiknya lewat operasi
// khusus (misalnya khusus admin), bukan lewat update profil biasa.
func (r *userPostgresRepository) Update(ctx context.Context, u model.User) (model.User, error) {
	err := scanUser(
		r.pool.QueryRow(ctx,
			`UPDATE users SET username = $1, email = $2, is_active = $3 WHERE id = $4 RETURNING `+kolomUser,
			u.Username, u.Email, u.IsActive, u.ID,
		),
		&u,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return model.User{}, ErrDuplicate
		}
		return model.User{}, fmt.Errorf("memperbarui user: %w", err)
	}
	return u, nil
}

func (r *userPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// isUniqueViolation memeriksa apakah error berasal dari pelanggaran
// batasan UNIQUE. Kode 23505 adalah kode resmi PostgreSQL untuk itu.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
