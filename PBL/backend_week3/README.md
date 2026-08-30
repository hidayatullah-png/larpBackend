# API Manajemen Data Mahasiswa

Proyek ini adalah RESTful API untuk mengelola data mahasiswa, dibangun menggunakan Go (Fiber) dan PostgreSQL. Ikuti panduan di bawah ini untuk menyiapkan dan menjalankan aplikasi dari awal di mesin lokalmu.

## 1. Persiapan Basis Data (Dari Nol)

Pastikan PostgreSQL sudah terinstal dan layanannya ( *service* ) sedang berjalan di komputermu.

1. Buka terminal (CMD/PowerShell) atau *tools* seperti pgAdmin/DBeaver.
2. Masuk ke PostgreSQL menggunakan *user* bawaan (biasanya `postgres`). Jika lewat terminal:
   ```bash
   psql -U postgres

habis tuh CREATE DATABASE
hbias tuh baut skema nya

CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    nim VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    grade DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS students_nim_lower_key ON students (LOWER(nim));

CREATE INDEX IF NOT EXISTS students_name_lower_idx ON students (LOWER(name));

habis tuh atur enviroment
# Konfigurasi Server API
APP_PORT=3000

# Konfigurasi PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=isi_dengan_password_postgres_kamu
DB_NAME=praktikum_backend
DB_SSLMODE=disable