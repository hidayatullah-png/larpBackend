package service

import (
	"strings"
	
	"latihan-fiber/app/model"
)

//File berisi business rule murnis(tidak menyentuh fiber.Ctx)
// tidak menyentuh database, dan tidak menyentuh HTTP request/response

func ValidateCreate(reqq *model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(reqq.NIM) == "" {
		errs["nim"] = "wajib diisi"
	}
	if strings.TrimSpace(reqq.Name) == "" {
		errs["name"] = "wajib diisi"
	}
	if reqq.Grade < 0 || reqq.Grade > 4 {
		errs["grade"] = "harus antara 0.0 sampai 4.0"
	}
	if len(errs) == 0 {
		return nil
	}	
	return errs
}

//ValidateRepalce memeriksa isi permintaan PUT, dan mengembalikan daftar error jika ada
func ValidateReplace(req *model.ReplaceStudentRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "wajib diisi pada PUT"
	}
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "wajib diisi pada PUT"
	}
	if req.Grade < 0 || req.Grade > 4 {
		errs["grade"] = "harus antara 0.0 sampai 4.0"
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

//ApplyPatch memeriksa isi permintaan PATCH, dan mengembalikan daftar error jika ada
func ApplyPatch(s *model.Student, req *model.PatchStudentRequest) map[string]string {
	errs := map[string]string{}
	
	if req.NIM != nil {
		if strings.TrimSpace(*req.NIM) == "" {
			errs["nim"] = "wajib diisi pada PATCH"
		} else {
			s.NIM = *req.NIM
		}
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errs["name"] = "wajib diisi pada PATCH"
		} else {
			s.Name = *req.Name
		}
	}
	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 4 {
			errs["grade"] = "harus antara 0.0 sampai 4.0"
		} else {
			s.Grade = *req.Grade
		}
	}

	// TAMBAHKAN BLOK INI UNTUK UPDATE IS_ACTIVE
	if req.IsActive != nil {
		s.IsActive = *req.IsActive
	}

	// Konsistenkan dengan fungsi Validate lainnya (return nil jika tidak ada error)
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// IsEmptyPatch memeriksa apakah permintaan PATCH kosong (tidak ada field yang diubah)
func IsEmptyPatch(req *model.PatchStudentRequest) bool {
	// TAMBAHKAN PENGECEKAN && req.IsActive == nil
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}

//CountTotalPages menghitung jumlah halaman total berdasarkan total data dan limit per halaman
func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}