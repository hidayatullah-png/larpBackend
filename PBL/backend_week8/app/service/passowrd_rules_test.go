package service

import "testing"

func TestIsStrongPassword(t *testing.T) {
	// Skenario 1: Password terlalu pendek (Harus Gagal / false)
	if IsStrongPassword("Ab1") {
		t.Errorf("Harusnya gagal karena terlalu pendek")
	}

	// Skenario 2: Password tanpa angka (Harus Gagal / false)
	if IsStrongPassword("PasswordTanpaAngka") {
		t.Errorf("Harusnya gagal karena tidak ada angka")
	}

	// Skenario 3: Password kuat dan valid (Harus Berhasil / true)
	if !IsStrongPassword("PasswordAman123") {
		t.Errorf("Harusnya lolos karena sudah memenuhi syarat")
	}
}
