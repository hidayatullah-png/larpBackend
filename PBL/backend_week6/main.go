package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"latihan-fiber/app/repository"
	"latihan-fiber/app/service"
	"latihan-fiber/config"
	"latihan-fiber/database"
	"latihan-fiber/helper"
	"latihan-fiber/route"
)

const minSecretLength = 32

func main() {
	config.LoadEnv()
	logger := config.NewLogger()

	// Pengecekan JWT Secret sebelum server menyala
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < minSecretLength {
		logger.Error("JWT_SECRET tidak diisi atau terlalu pendek",
			slog.Int("minimal_karakter", minSecretLength))
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	jwtManager := helper.NewJWTManager(jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	// 1. Inisialisasi Repository
	userRepository := repository.NewUserRepository(pool)
	tokenRepository := repository.NewTokenRepository(pool)
	studentRepository := repository.NewStudentRepository(pool)
	roleRepository := repository.NewRoleRepository(pool)

	// 2. Muat pemetaan role dan permission dari database sekali saja
	rawPermissions, err := roleRepository.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission", slog.String("error", err.Error()))
		os.Exit(1) // Fail closed: aplikasi menolak menyala jika izin gagal dimuat
	}

	permissions := helper.NewPermissionSet(rawPermissions)
	logger.Info("permission dimuat", slog.Any("roles", permissions.KnownRoles()))

	// 3. Inisialisasi Service
	// Catatan: Jika NewUserService/NewAuthService belum diubah untuk menerima
	// parameter 'permissions', mungkin perlu menambahkan argument tersebut di file service masing-masing agar tidak error.
	userService := service.NewUserService(userRepository)
	authService := service.NewAuthService(
		userRepository, tokenRepository, jwtManager,
		permissions, // <- TAMBAHKAN VARIABEL INI DI SINI
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour,
	)
	studentService := service.NewStudentService(studentRepository, permissions)

	// 4. Daftarkan ke Dependencies
	app := config.NewApp(logger, route.Dependencies{
		Pool:           pool,
		JWT:            jwtManager,
		Permissions:    permissions,
		UserService:    userService,
		AuthService:    authService,
		StudentService: studentService,
	})

	port := config.GetEnv("APP_PORT", "3000")

	// Jalankan server di dalam goroutine
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("gagal menjalankan server", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("server berjalan", slog.String("port", port))

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("memulai proses mematikan server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("gagal mematikan server", slog.String("error", err.Error()))
	}

	logger.Info("server berhasil dimatikan dengan aman")
}
