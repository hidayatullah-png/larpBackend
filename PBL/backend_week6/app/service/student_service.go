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

	// 2. Pemetaan dan Penyuntikan OwnerID secara paksa.
	// Ini menutup celah Mass Assignment, di mana peretas tidak bisa melakukan fraud
	// mendaftarkan data atas nama orang lain.
	studentData := model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		IsActive: true,
		OwnerID:  int(current.UserID),
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
	// Periksa apakah user ini pemiliknya, atau punya hak :any
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

	// Route ini diurus oleh Middleware,
	// jadi siapa pun yang sampai ke titik ini dipastikan sudah punya izin (admin).
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Fail(c, fiber.StatusNotFound, "data mahasiswa tidak ditemukan")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal menghapus data")
	}

	return helper.NoContent(c)
}

// List (GET /students)
func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	students, err := s.repo.FindAll(ctx)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}
	return helper.Success(c, fiber.StatusOK, "berhasil", students)
}

// Replace (PUT /students/:id)
func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, _ := helper.CurrentUser(c)
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id invalid")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body invalid")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return helper.Fail(c, fiber.StatusNotFound, "data tidak ditemukan")
	}

	// Periksa apakah ia pemiliknya atau punya hak update:any
	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "tidak berhak mengubah data ini")
	}

	student.NIM = req.NIM
	student.Name = req.Name
	student.IsActive = req.IsActive

	updated, err := s.repo.Update(ctx, student)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengubah data")
	}
	return helper.Success(c, fiber.StatusOK, "berhasil", updated)
}

// Patch (PATCH /students/:id)
func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, _ := helper.CurrentUser(c)
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id invalid")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body invalid")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return helper.Fail(c, fiber.StatusNotFound, "data tidak ditemukan")
	}

	// TUGAS MANDIRI C.2 POIN 3: Periksa apakah ia pemiliknya atau punya hak update:any
	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "tidak berhak mengubah data ini")
	}

	if req.NIM != nil {
		student.NIM = *req.NIM
	}
	if req.Name != nil {
		student.Name = *req.Name
	}
	if req.IsActive != nil {
		student.IsActive = *req.IsActive
	}

	updated, err := s.repo.Update(ctx, student)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengubah data")
	}
	return helper.Success(c, fiber.StatusOK, "berhasil", updated)
}
