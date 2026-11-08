package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/app/service"
	"latihan-fiber/config"
	"latihan-fiber/helper"
	"latihan-fiber/route"
)

// api adalah aplikasi lengkap yang memakai PostgreSQL sungguhan. Bedanya
// dengan production hanya satu: database-nya praktikum_backend_test.
type api struct {
	app *fiber.App
	jwt *helper.JWTManager
}

func newAPI(t *testing.T) api {
	t.Helper()
	raw, err := repository.NewRoleRepository(pool).LoadPermissions(context.Background())
	if err != nil {
		t.Fatalf("memuat permission: %v", err)
	}
	perms := helper.NewPermissionSet(raw)
	jwtManager := helper.NewJWTManager("secret-khusus-test-minimal-32-karakter", "test", time.Minute)

	courseService := service.NewCourseService(
		repository.NewCourseRepository(pool),
		repository.NewOutboxRepository(pool),
		repository.NewTxManager(pool),
	)

	app := config.NewApp(slog.New(slog.NewTextHandler(io.Discard, nil)), route.Dependencies{
		Pool:          pool,
		JWT:           jwtManager,
		Permissions:   perms,
		CourseService: courseService,
	})

	return api{app: app, jwt: jwtManager}
}

// enroll mengirim POST /courses/:id/enroll atas nama student dan
// mengembalikan status code beserta field "code" pada response.
func (a api) enroll(t *testing.T, student model.User, courseID int) (int, string) {
	t.Helper()
	token, err := a.jwt.GenerateAccess(model.User{ID: student.ID, Username: student.Username, Role: "user"})
	if err != nil {
		t.Errorf("membuat token: %v", err)
		return 0, ""
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+strconv.Itoa(courseID)+"/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		// t.Errorf, bukan t.Fatalf: fungsi ini juga dipanggil dari
		// goroutine, dan t.Fatalf hanya boleh dipanggil dari goroutine test.
		t.Errorf("request enroll: %v", err)
		return 0, ""
	}
	defer resp.Body.Close()

	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.Code
}

func countRows(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	return n
}

func TestEnrollEndpoint_WritesOutboxOnlyOnSuccess(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)
	course := seedCourse(t, 3, 40)

	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran pertama: status = %d, seharusnya 201", status)
	}

	if status, code := a.enroll(t, student, course.ID); status != http.StatusConflict || code != "ALREADY_ENROLLED" {
		t.Fatalf("pendaftaran kedua: dapat %d %s, seharusnya 409 ALREADY_ENROLLED", status, code)
	}

	events := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentCreated)

	if events != 1 {
		t.Errorf("event enrollment.created = %d, seharusnya 1 (pendaftaran yang gagal tidak boleh meninggalkan jejak)", events)
	}
}

// Satu mahasiswa mengirim sepuluh pendaftaran SERENTAK ke sepuluh kelas
// 3 SKS. Batasnya 24 SKS, maka tepat delapan yang boleh berhasil.
// Test ini adalah versi otomatis dari uji beban mode credits pada Modul 9.
func TestEnrollEndpoint_ConcurrentCreditLimit(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)
	const attempts = 10
	courses := make([]model.Course, attempts)
	for i := range courses {
		courses[i] = seedCourse(t, 3, 40)
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = map[int]int{}
		start   = make(chan struct{})
	)

	for _, c := range courses {
		wg.Add(1)
		go func(courseID int) {
			defer wg.Done()
			<-start // tunggu aba-aba agar semua request berangkat bersamaan
			status, _ := a.enroll(t, student, courseID)
			mu.Lock()
			results[status]++
			mu.Unlock()
		}(c.ID)
	}
	close(start) // aba-aba: semua goroutine dilepas pada saat yang sama
	wg.Wait()

	maxCourses := service.MaxCreditsPerSemester / 3
	if results[http.StatusCreated] != maxCourses {
		t.Errorf("berhasil = %d, seharusnya %d (hasil lengkap: %v)", results[http.StatusCreated], maxCourses, results)
	}

	enrolled := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1`, student.ID)
	if enrolled != maxCourses {
		t.Errorf("baris enrollments = %d, seharusnya %d", enrolled, maxCourses)
	}
}

func (a api) drop(t *testing.T, student model.User, courseID int) int {
	t.Helper()
	token, _ := a.jwt.GenerateAccess(model.User{ID: student.ID, Username: student.Username, Role: "user"})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/"+strconv.Itoa(courseID)+"/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Errorf("request drop: %v", err)
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func (a api) swap(t *testing.T, student model.User, reqBody map[string]int) (int, string) {
	t.Helper()
	token, _ := a.jwt.GenerateAccess(model.User{ID: student.ID, Username: student.Username, Role: "user"})

	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments/swap", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Errorf("request swap: %v", err)
		return 0, ""
	}
	defer resp.Body.Close()

	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.Code
}

func TestEnrollEndpoint_SwapTargetFull(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)
	dummyStudent := seedStudent(t) // Mahasiswa tumbal untuk memenuhi kelas

	courseA := seedCourse(t, 2, 40)
	courseB := seedCourse(t, 2, 1) // Kuota 1 agar lolos pengecekan database

	// Penuhi kelas B dengan mahasiswa tumbal (sekarang kelas B penuh 1/1)
	a.enroll(t, dummyStudent, courseB.ID)

	// Persiapan: mahasiswa utama sudah terdaftar di Kelas A
	a.enroll(t, student, courseA.ID)

	req := map[string]int{"from_course_id": courseA.ID, "to_course_id": courseB.ID}
	status, code := a.swap(t, student, req)

	if status != http.StatusConflict || code != "COURSE_FULL" {
		t.Errorf("swap ke kelas penuh: dapat %d %s, seharusnya 409 COURSE_FULL", status, code)
	}

	enrolledA := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student.ID, courseA.ID)
	if enrolledA != 1 {
		t.Error("karena gagal, enrollment asal (Kelas A) harusnya tetap dipertahankan")
	}
}

func TestEnrollEndpoint_DropSuccess(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)
	course := seedCourse(t, 3, 40)

	// Persiapan: daftar dulu
	a.enroll(t, student, course.ID)

	status := a.drop(t, student, course.ID)
	if status != http.StatusNoContent {
		t.Fatalf("drop kelas: dapat status %d, seharusnya 204", status)
	}

	enrolled := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student.ID, course.ID)
	if enrolled != 0 {
		t.Error("data enrollment seharusnya terhapus")
	}

	events := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentDropped) 

	if events != 1 {
		t.Error("event course.dropped seharusnya tercatat di outbox")
	}
}