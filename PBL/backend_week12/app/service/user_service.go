package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// UserService memegang dua tanggung jawab sekaligus pada struktur baku
// mata kuliah ini: menerima *fiber.Ctx (peran controller) dan menjalankan
// business rules (peran use case)
type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// List (GET /users) - Menggunakan Cursor Pagination & Content Negotiation
func (s *UserService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// Format dipilih SEBELUM query dijalankan agar tidak membuang resource DB
	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	if format == helper.FormatCSV {
		return helper.WriteUsersCSV(c, rows)
	}

	// Baris tambahan hasil limit+1 dipotong di sini
	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar user berhasil diambil", rows, meta)
}

func (s *UserService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "user ditemukan", user)
}

func (s *UserService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	// Validasi deklaratif terpusat
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	newUser, err := s.repo.Create(ctx, model.User{
		Username: strings.TrimSpace(req.Username),
		Email:    strings.TrimSpace(req.Email),
		Password: req.Password,
		IsActive: true,
	})
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.Created(c, "user berhasil dibuat", newUser, "/api/v1/users/"+strconv.Itoa(newUser.ID))
}

func (s *UserService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.ReplaceUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	// Validasi deklaratif terpusat
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	result, err := s.repo.Update(ctx, model.User{
		ID:       id,
		Username: strings.TrimSpace(req.Username),
		Email:    strings.TrimSpace(req.Email),
		IsActive: req.IsActive,
	})
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "user berhasil diganti seluruhnya", result)
}

func (s *UserService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.PatchUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Pengecekan inline agar tidak perlu fungsi IsEmptyPatch dari luar
	if req.Username == nil && req.Email == nil && req.IsActive == nil {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateUserError(err, "user")
	}

	updated := current

	// Terapkan data baru hanya jika field-nya dikirim (tidak nil)
	if req.Username != nil {
		updated.Username = strings.TrimSpace(*req.Username)
	}
	if req.Email != nil {
		updated.Email = strings.TrimSpace(*req.Email)
	}
	if req.IsActive != nil {
		updated.IsActive = *req.IsActive
	}

	result, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateUserError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "user berhasil diperbarui sebagian", result)
}

func (s *UserService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateUserError(err, "user")
	}

	return helper.NoContent(c)
}

// translateUserError mengubah error milik repository menjadi AppError
func translateUserError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("username sudah dipakai")
	default:
		// Mengembalikan error internal untuk dicatat di log tanpa membocorkan detail SQL ke client
		return helper.Internal(err)
	}
}
