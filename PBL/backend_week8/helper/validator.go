package helper

import (
	"errors"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// validate dibuat SEKALI untuk seluruh aplikasi
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()

	// Tanpa ini, pesan error menyebut nama field Go ("Username"), padahal client mengirim dan membaca nama JSON ("username")
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})

	// Aturan buatan sendiri
	_ = v.RegisterValidation("nospace", func(fl validator.FieldLevel) bool {
		return !strings.ContainsAny(fl.Field().String(), " \t\n\r")
	})

	_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		for _, r := range fl.Field().String() {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '_' {
				return false
			}
		}
		return true
	})

	// BUG 5 DIPERBAIKI: Harus == "" agar password kuat diterima
	_ = v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		return passwordStrength(fl.Field().String()) == "" 
	})

	// TUGAS D.2: Aturan custom untuk NIM mahasiswa
	_ = v.RegisterValidation("nim_ti", func(fl validator.FieldLevel) bool {
		nim := fl.Field().String()
		return strings.HasPrefix(nim, "434")
	})

	return v
}

// ValidateStruct menjalankan seluruh aturan pada tag struct dan mengembalikan peta nama field ke pesan berbahasa Indonesia
func ValidateStruct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return map[string]string{"_": "objek yang divalidasi tidak sah"}
	}

	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return map[string]string{"_": "validasi gagal"}
	}

	result := make(map[string]string, len(fieldErrors))
	for _, fe := range fieldErrors {
		if _, exists := result[fe.Field()]; !exists {
			result[fe.Field()] = messageFor(fe)
		}
	}
	return result
}

// messageFor menerjemahkan nama tag menjadi kalimat yang dapat dibaca pemakai.
func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		if fe.Kind() == reflect.String {
			return "minimal " + fe.Param() + " karakter"
		}
		return "nilai minimal " + fe.Param()
	case "max":
		if fe.Kind() == reflect.String {
			return "maksimal " + fe.Param() + " karakter"
		}
		return "nilai maksimal " + fe.Param()
	case "len":
		return "harus tepat " + fe.Param() + " karakter"
	case "numeric":
		return "hanya boleh berisi angka"
	case "alphanum":
		return "hanya boleh berisi huruf dan angka"
	case "nospace":
		return "tidak boleh mengandung spasi"
	case "username":
		return "hanya boleh huruf, angka, titik, dan garis bawah"
	case "strongpassword":
		if value, ok := fe.Value().(string); ok {
			return passwordStrength(value)
		}
		return "password tidak memenuhi syarat"
	case "nim_ti": // TUGAS D.2: Pesan error untuk NIM
		return "format NIM tidak valid, harus diawali dengan kode prodi (misal: 434)"
	case "oneof":
		return "harus salah satu dari: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	default:
		return "tidak memenuhi aturan " + fe.Tag()
	}
}

// Dummy function passwordStrength agar tidak error, sesuaikan jika ada logika aslinya di auth_rules.go
func passwordStrength(pwd string) string {
	if len(pwd) < 8 {
		return "minimal 8 karakter"
	}
	// Tambahkan logika validasi password aslimu di sini
	return ""
}