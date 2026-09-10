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