package service

import (
	"latihan-fiber/app/model"
	"latihan-fiber/helper"
)

// CanAccessStudent memutuskan apakah seseorang boleh menyentuh data student tertentu.
// Memenuhi Tugas Mandiri C.2 poin 2: Fungsi murni tanpa import fiber/repository.
func CanAccessStudent(
	current model.AuthUser,
	ownerID int64,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	// 1. Jalur Kepemilikan (Ownership): Data miliknya sendiri
	// Karena UserID tipenya int dan ownerID int64, kita lakukan casting ke int64 agar tidak error.
	if int64(current.UserID) == ownerID {
		return true
	}

	// 2. Jalur Hak Akses: Role-nya memang berhak atas data siapa pun (permission :any)
	return perms.Can(current.Role, anyPermission)
}