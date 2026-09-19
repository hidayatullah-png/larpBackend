package service

import (
	"errors"
	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"

	"github.com/gofiber/fiber/v2"
)

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, perms: perms}
}

// Create (POST /students)
func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// 1. Ambil identitas pemanggil dari token JWT
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	// 2. Pemetaan dan Penyuntikan OwnerID secara paksa
	// Menjawab C.2 Poin 4: owner_id TIDAK DIAMBIL DARI BODY JSON, melainkan dari current.UserID.
	// Ini menutup celah Mass Assignment, di mana peretas tidak bisa mengaku-aku 
	// mendaftarkan data atas nama orang lain.
	studentData := model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		IsActive: true,
		OwnerID:  int64(current.UserID),
	}

	created, err := s.repo.Insert(ctx, studentData)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan data mahasiswa")
	}

	return helper.Success(c, fiber.StatusCreated, "data mahasiswa berhasil ditambahkan", created)
}

// Get (GET /students/:id)
func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	// 1. AMBIL DATANYA DULU DARI DATABASE
	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Fail(c, fiber.StatusNotFound, "data mahasiswa tidak ditemukan")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	// 2. PERIKSA HAK AKSES SETELAH DATA DIAMBIL
	// Menjawab C.2 Poin 3: Periksa apakah user ini pemiliknya, atau punya hak :any
	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "anda tidak berhak mengakses data mahasiswa lain")
	}

	return helper.Success(c, fiber.StatusOK, "data ditemukan", student)
}

// Delete (DELETE /students/:id)
func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	// Sesuai C.2 Poin 1: Route ini diurus oleh Middleware, 
	// jadi siapa pun yang sampai ke titik ini dipastikan sudah punya izin (admin).
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Fail(c, fiber.StatusNotFound, "data mahasiswa tidak ditemukan")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal menghapus data")
	}

	return helper.NoContent(c)
}