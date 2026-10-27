-- Migration 007: pembimbing tugas akhir dan bimbingan (Tugas Mandiri D.2)
CREATE TABLE IF NOT EXISTS supervisors (
    id               SERIAL       PRIMARY KEY,
    nidn             VARCHAR(20)  NOT NULL UNIQUE,
    name             VARCHAR(100) NOT NULL,
    quota            INTEGER      NOT NULL CHECK (quota > 0),
    supervised_count INTEGER      NOT NULL DEFAULT 0,
    version          INTEGER      NOT NULL DEFAULT 1,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT supervisors_count_within_quota
        CHECK (supervised_count >= 0 AND supervised_count <= quota)
);

CREATE TABLE IF NOT EXISTS supervisions (
    id            BIGSERIAL   PRIMARY KEY,
    supervisor_id INTEGER     NOT NULL REFERENCES supervisors(id) ON DELETE RESTRICT,
    student_id    INTEGER     NOT NULL REFERENCES students(id)    ON DELETE CASCADE,
    assigned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Satu mahasiswa hanya boleh memiliki satu pembimbing aktif
    CONSTRAINT supervisions_student_unique UNIQUE (student_id)
);

CREATE INDEX IF NOT EXISTS idx_supervisions_supervisor_id ON supervisions(supervisor_id);

INSERT INTO supervisors (nidn, name, quota) VALUES
    ('0011018501', 'Dr. Budi Raharjo, M.Kom.',     20),
    ('0022038702', 'Dr. Siti Aminah, S.T., M.T.',  20),
    ('0005089003', 'Prof. Hendra Wijaya, Ph.D.',    3)
ON CONFLICT (nidn) DO NOTHING;

INSERT INTO permissions (name, description) VALUES
    ('supervision:manage', 'Mengelola kuota dan penetapan dosen pembimbing')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'supervision:manage'),
    ('user',  'supervision:manage')
ON CONFLICT DO NOTHING;

-- Pastikan minimal tersedia 35 data students untuk uji beban D.3
INSERT INTO students (nim, name, grade, owner_id)
SELECT
    '1999' || LPAD(i::text, 4, '0'),
    'Mahasiswa TA ' || LPAD(i::text, 2, '0'),
    3.50,
    (SELECT id FROM users ORDER BY id LIMIT 1)
FROM generate_series(1, 35) AS s(i)
WHERE NOT EXISTS (
    SELECT 1 FROM students WHERE nim = '1999' || LPAD(i::text, 4, '0')
);