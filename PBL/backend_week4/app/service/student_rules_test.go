package service

import (
	"testing"
	"latihan-fiber/app/model"
)

func TestCountTotalPages(t *testing.T) {
	cases := []struct{ total, limit, want int }{
		{0, 10, 0},
		{1, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{137, 20, 7},
	}

	for _, tc := range cases {
		if got := CountTotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("total=%d limit=%d: harap %d, dapat %d",
				tc.total, tc.limit, tc.want, got)
		}
	}
}

func TestApplyPatch(t *testing.T) {
	// 1. Persiapkan data awal simulasi
	initial := model.Student{ID: 1, NIM: "0812345", Name: "sari", Grade: 3.8, IsActive: true}
	inactive := false

	// 2. Jalankan fungsi dari student_rules.go
	errs := ApplyPatch(&initial, &model.PatchStudentRequest{IsActive: &inactive})

	// 3. Evaluasi hasilnya
	if len(errs) != 0 {
		t.Fatalf("tidak seharusnya ada error: %v", errs)
	}
	if initial.IsActive {
		t.Error("is_active seharusnya berubah menjadi false")
	}
	if initial.Name != "sari" {
		t.Error("field yang tidak dikirim seharusnya tidak berubah")
	}
}