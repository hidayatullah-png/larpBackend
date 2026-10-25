CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    nim VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    grade DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    mata_kuliah VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS students_nim_lower_key ON students (LOWER(nim));

CREATE INDEX IF NOT EXISTS students_name_lower_idx ON students (LOWER(name));

ALTER TABLE students DROP COLUMN mata_kuliah, DROP COLUMN grade;

CREATE TABLE IF NOT EXISTS nilai (
    id_nilai SERIAL PRIMARY KEY,
    id_student INT NOT NULL,
    mata_kuliah VARCHAR(100) NOT NULL,
    grade DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    CONSTRAINT fk_student FOREIGN KEY (id_student) REFERENCES students(id) ON DELETE CASCADE
);

-- 1. Buat tabel users jika belum ada dari pertemuan sebelumnya
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Tambahkan kolom role untuk otorisasi[cite: 1]
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

-- 3. Tabel refresh_tokens disimpan sebagai HASH[cite: 1]
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         BIGSERIAL   PRIMARY KEY,
    user_id    INTEGER     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Index untuk mempercepat pencarian berdasarkan user_id[cite: 1]
CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx
    ON refresh_tokens (user_id);