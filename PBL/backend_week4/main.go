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
)

func main() {
	// 1. Konfigurasi dan inisialisasi logger
	config.LoadEnv()
	logger := config.NewLogger()

	// 2. Database
	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// 3. Perakitan dari dalam ke luar: repository -> service
	// Perbaikan: Ganti NewUserRepository menjadi NewStudentRepository
	studentRepository := repository.NewStudentRepository(pool)
	// Perbaikan: Ganti NewUserService menjadi NewStudentService
	studentService := service.NewStudentService(studentRepository)

	// 4. Aplikasi
	app := config.NewApp(logger, pool, *studentService)

	port := config.GetEnv("APP_PORT", "3000")

	// Jalankan server di dalam goroutine agar tidak memblokir proses di bawahnya
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("gagal menjalankan server", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("server berjalan", slog.String("port", port))

	// 5. Graceful shutdown: tunggu Ctrl+C, lalu beri waktu request selesai
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