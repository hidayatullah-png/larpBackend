package service

import (
	"latihan-fiber/app/model"
	"latihan-fiber/helper"
)

func CanAccessStudent(
	current model.AuthUser,
	ownerID int, 
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	// Karena tipe datanya sekarang sudah sama-sama int, bisa langsung mdibandingkan.
	if current.UserID == ownerID {
		return true
	}

	return perms.Can(current.Role, anyPermission)
}