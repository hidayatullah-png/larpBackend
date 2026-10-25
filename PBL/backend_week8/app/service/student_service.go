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

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	studentData := model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade, // Atribut Grade dipetakan
		IsActive: true,
		OwnerID:  int(current.UserID),
	}

	created, err := s.repo.Insert(ctx, studentData)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusCreated, "data mahasiswa berhasil ditambahkan", created)
}

// Get (GET /students/:id)
func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	_, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "data mahasiswa ditemukan", student)
}

// Delete (DELETE /students/:id)
func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.NoContent(c)
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// 1. Negosiasi format SEBELUM query dijalankan[cite: 43].
	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	// 2. Persiapkan Cursor (Anda bisa memisahkan parsing URL ke helper terpisah)
	limit := c.QueryInt("limit", 10)
	cursorStr := c.Query("cursor")

	q := model.CursorQuery{Limit: limit}

	if cursorStr != "" {
		parsedCursor, err := helper.DecodeCursor(cursorStr)
		if err != nil {
			return err
		}
		q.After = &parsedCursor
	}

	// 3. Ambil data dari repository
	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	// 4. Kirim sebagai CSV jika formatnya diminta[cite: 43]
	if format == helper.FormatCSV {
		// Baris tambahan limit+1 TIDAK dipotong untuk CSV
		return helper.WriteStudentsCSV(c, rows)
	}

	// 5. Eksekusi logika Cursor khusus JSON
	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit] // Potong baris tambahan hasil limit+1[cite: 42].
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar mahasiswa berhasil diambil", rows, meta)
}

// Replace (PUT /students/:id)
func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id invalid")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body invalid")
	}

	// Validasi input
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data ini")
	}

	student.NIM = req.NIM
	student.Name = req.Name
	student.Grade = req.Grade // Atribut Grade dipetakan
	student.IsActive = req.IsActive

	updated, err := s.repo.Update(ctx, student)
	if err != nil {
		return translateError(err, "mahasiswa")
	}
	return helper.Success(c, fiber.StatusOK, "berhasil", updated)
}

// Patch (PATCH /students/:id)
func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id invalid")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body invalid")
	}

	// Validasi input
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data ini")
	}

	if req.NIM != nil {
		student.NIM = *req.NIM
	}
	if req.Name != nil {
		student.Name = *req.Name
	}
	if req.Grade != nil {
		student.Grade = *req.Grade // Atribut Grade dipetakan
	}
	if req.IsActive != nil {
		student.IsActive = *req.IsActive
	}

	updated, err := s.repo.Update(ctx, student)
	if err != nil {
		return translateError(err, "mahasiswa")
	}
	return helper.Success(c, fiber.StatusOK, "berhasil", updated)
}

// translateError mengubah error milik repository menjadi AppError.
func translateError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict(entity + " sudah ada")
	default:
		// Bug 8 diperbaiki: Error dari DB dicatat sebagai 500 Internal Server Error
		return helper.Internal(err)
	}
}
