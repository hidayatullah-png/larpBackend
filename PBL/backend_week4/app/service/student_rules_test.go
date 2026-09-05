package service

import (
	"testing"
	"latihan-fiber/app/model"
)

// 1. Test Validasi POST (Create)
func TestValidateCreate(t *testing.T) {
	req := &model.CreateStudentRequest{NIM: "", Name: "Budi", Grade: 5.0}
	errs := ValidateCreate(req)
	
	if errs == nil {
		t.Fatal("seharusnya mengembalikan error karena NIM kosong dan Grade > 4")
	}
	if errs["nim"] == "" {
		t.Error("error nim tidak terdeteksi")
	}
	if errs["grade"] == "" {
		t.Error("error grade tidak terdeteksi")
	}
}

// 2. Test Validasi PUT (Replace)
func TestValidateReplace(t *testing.T) {
	req := &model.ReplaceStudentRequest{NIM: "123", Name: "", Grade: 3.5, IsActive: true}
	errs := ValidateReplace(req)
	
	if errs == nil {
		t.Fatal("seharusnya mengembalikan error karena Name kosong")
	}
	if errs["name"] == "" {
		t.Error("error name tidak terdeteksi")
	}
}

// 3. Test Penerapan PATCH
func TestApplyPatch(t *testing.T) {
	initial := model.Student{ID: 1, NIM: "0812345", Name: "Sari", Grade: 3.8, IsActive: true}
	inactive := false
	newName := "Sari Indah"

	errs := ApplyPatch(&initial, &model.PatchStudentRequest{
		Name:     &newName,
		IsActive: &inactive,
	})

	if errs != nil {
		t.Fatalf("tidak seharusnya ada error: %v", errs)
	}
	if initial.IsActive {
		t.Error("is_active seharusnya berubah menjadi false")
	}
	if initial.Name != "Sari Indah" {
		t.Error("name seharusnya berubah menjadi Sari Indah")
	}
	if initial.Grade != 3.8 {
		t.Error("grade yang tidak dikirim seharusnya tidak berubah")
	}
}