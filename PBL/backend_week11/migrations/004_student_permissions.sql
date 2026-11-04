-- 1. Tambahkan daftar permission baru untuk entitas students
INSERT INTO permissions (name, description) VALUES
('student:list', 'Melihat daftar mahasiswa'),
('student:read:any', 'Melihat data mahasiswa mana pun'),
('student:create', 'Menambah data mahasiswa'),
('student:update:any', 'Mengubah data mahasiswa mana pun'),
('student:delete', 'Menghapus mahasiswa')
ON CONFLICT (name) DO NOTHING;

-- 2. Pasangkan permission tersebut ke role admin dan staff
INSERT INTO role_permissions (role_name, permission_name) VALUES
('admin', 'student:list'),
('admin', 'student:read:any'),
('admin', 'student:create'),
('admin', 'student:update:any'),
('admin', 'student:delete'),
('staff', 'student:list'),
('staff', 'student:read:any'),
('staff', 'student:create')
ON CONFLICT DO NOTHING;

-- 3. Tambahkan kolom owner_id pada tabel students tanpa batasan NOT NULL terlebih dahulu
ALTER TABLE students ADD COLUMN IF NOT EXISTS owner_id BIGINT;

-- 4. Amankan data lama: Set owner_id untuk baris students yang sudah ada (misal diserahkan ke user ID 1)
UPDATE students SET owner_id = 1 WHERE owner_id IS NULL;

-- 5. Kunci kolom owner_id menjadi NOT NULL dan buat constraint FOREIGN KEY ke tabel users
ALTER TABLE students ALTER COLUMN owner_id SET NOT NULL;
ALTER TABLE students DROP CONSTRAINT IF EXISTS fk_student_owner;
ALTER TABLE students ADD CONSTRAINT fk_student_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;