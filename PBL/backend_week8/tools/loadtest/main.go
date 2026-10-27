package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"latihan-fiber/app/model"
	"latihan-fiber/config"
	"latihan-fiber/database"
	"latihan-fiber/helper"
)

type testAccount struct {
	User  model.User
	Token string
}

// fire menjalankan n pekerjaan secara serentak dan menghitung hasilnya.
//
// Setiap goroutine menunggu di depan "gerbang" start yang sama. Gerbang
// baru dibuka setelah SELURUH goroutine siap, sehingga request benar-benar
// berangkat hampir bersamaan bukan bergiliran sesuai urutan dibuatnya.
func fire(n int, job func(i int) string) map[string]int {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		// PERBAIKAN BUG #2: Ubah "out <-chan string" menjadi "out chan<- string"
		go func(i int, out chan<- string) {
			defer wg.Done()
			<-start
			out <- job(i)
		}(i, results)
	}

	close(start) // buka gerbang
	wg.Wait()    // tunggu SEMUA goroutine selesai
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally
}

func main() {
	mode := flag.String("mode", "crowd", "mode uji: crowd | credits | swap")
	baseURL := flag.String("url", "http://localhost:3000/api/v1", "base URL API")
	flag.Parse()

	config.LoadEnv()
	ctx := context.Background()

	pool, err := database.NewPool(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal terhubung ke database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	jwtManager := helper.NewJWTManager(
		config.GetEnv("JWT_SECRET", ""),
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		15*time.Minute,
	)

	accounts, err := loadAccounts(ctx, pool, jwtManager)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal memuat akun uji: %v\n", err)
		os.Exit(1)
	}
	if len(accounts) < 60 {
		fmt.Fprintf(os.Stderr, "akun uji kurang dari 60 (ditemukan %d). Jalankan migrations/006_seed_loadtest.sql dulu.\n", len(accounts))
		os.Exit(1)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	switch *mode {
	case "crowd":
		runCrowd(ctx, pool, client, *baseURL, accounts)
	case "credits":
		runCredits(ctx, pool, client, *baseURL, accounts)
	case "swap":
		runSwap(ctx, pool, client, *baseURL, accounts)
	case "supervisor":
		runSupervisor(ctx, pool, client, *baseURL, accounts)
	default:
		fmt.Fprintf(os.Stderr, "mode tidak dikenal: %s (gunakan crowd, credits, atau swap)\n", *mode)
		os.Exit(1)
	}
}

func loadAccounts(ctx context.Context, pool *pgxpool.Pool, jwtManager *helper.JWTManager) ([]testAccount, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, username, email, role, is_active, created_at
		FROM users
		WHERE username LIKE 'mhs%'
		ORDER BY username ASC
		LIMIT 60`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []testAccount
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, err
		}
		// Alat ini mencetak token sendiri karena ia memegang JWT_SECRET.
		token, err := jwtManager.GenerateAccess(u)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, testAccount{User: u, Token: token})
	}
	return accounts, rows.Err()
}

func doRequest(client *http.Client, method, url, token string, body any) string {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return "ERR_REQ"
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "ERR_NET: " + err.Error()
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("%d OK", resp.StatusCode)
	}

	var errResp struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(raw, &errResp)
	if errResp.Code != "" {
		return fmt.Sprintf("%d %s", resp.StatusCode, errResp.Code)
	}
	return fmt.Sprintf("%d", resp.StatusCode)
}

func printTally(tally map[string]int, elapsed time.Duration) {
	fmt.Println("hasil request:")
	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-28s %d\n", k, tally[k])
	}
	fmt.Printf("waktu total: %dms\n", elapsed.Milliseconds())
}

func checkCourseConsistency(ctx context.Context, pool *pgxpool.Pool, code string) {
	var id, quota, enrolled int
	err := pool.QueryRow(ctx,
		`SELECT id, quota, enrolled_count FROM courses WHERE code = $1`, code,
	).Scan(&id, &quota, &enrolled)
	if err != nil {
		fmt.Printf("gagal memeriksa kelas %s: %v\n", code, err)
		return
	}

	var actual int
	_ = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM enrollments WHERE course_id = $1`, id,
	).Scan(&actual)

	status := "KONSISTEN"
	if enrolled != actual || enrolled > quota || actual > quota {
		status = "TIDAK KONSISTEN"
	}
	fmt.Printf("%s kuota=%d enrolled_count=%d baris enrollments=%d -> %s\n",
		code, quota, enrolled, actual, status)
}

// Mode 1: 60 mahasiswa berebut kelas IF399 (kuota 5)
func runCrowd(ctx context.Context, pool *pgxpool.Pool, client *http.Client, baseURL string, accounts []testAccount) {
	var courseID int
	err := pool.QueryRow(ctx, `SELECT id FROM courses WHERE code = 'IF399'`).Scan(&courseID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kelas IF399 tidak ditemukan: %v\n", err)
		return
	}

	fmt.Println("60 mahasiswa berebut kelas IF399 secara serentak...")
	url := fmt.Sprintf("%s/courses/%d/enroll", baseURL, courseID)

	start := time.Now()
	tally := fire(60, func(i int) string {
		return doRequest(client, http.MethodPost, url, accounts[i].Token, nil)
	})
	elapsed := time.Since(start)

	printTally(tally, elapsed)
	checkCourseConsistency(ctx, pool, "IF399")
}

// Mode 2: 10 mahasiswa masing-masing mendaftar ke 12 kelas 3 SKS secara serentak (batas 24 SKS)
func runCredits(ctx context.Context, pool *pgxpool.Pool, client *http.Client, baseURL string, accounts []testAccount) {
	rows, err := pool.Query(ctx, `SELECT id FROM courses WHERE code <> 'IF399' ORDER BY code LIMIT 12`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal mengambil 12 kelas: %v\n", err)
		return
	}
	defer rows.Close()

	var courseIDs []int
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		courseIDs = append(courseIDs, id)
	}

	fmt.Println("10 mahasiswa, masing-masing mendaftar ke 12 kelas 3 SKS secara serentak (batas 24 SKS)...")
	totalJobs := 10 * len(courseIDs)

	start := time.Now()
	tally := fire(totalJobs, func(i int) string {
		studentIdx := i / len(courseIDs)
		courseIdx := i % len(courseIDs)
		url := fmt.Sprintf("%s/courses/%d/enroll", baseURL, courseIDs[courseIdx])
		return doRequest(client, http.MethodPost, url, accounts[studentIdx].Token, nil)
	})
	elapsed := time.Since(start)

	printTally(tally, elapsed)

	// Periksa apakah ada mahasiswa yang melebihi 24 SKS
	overLimit := 0
	for i := 0; i < 10; i++ {
		var sks int
		_ = pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(c.credits), 0)
			FROM enrollments e
			JOIN courses c ON c.id = e.course_id
			WHERE e.user_id = $1`, accounts[i].User.ID).Scan(&sks)
		if sks > 24 {
			overLimit++
		}
	}

	if overLimit == 0 {
		fmt.Println("seluruh 10 mahasiswa <= 24 SKS -> KONSISTEN")
	} else {
		fmt.Printf("%d dari 10 mahasiswa melebihi 24 SKS -> TIDAK KONSISTEN\n", overLimit)
	}
}

// Mode 3: 20 pasang mahasiswa pindah kelas berlawanan arah (IF101 <-> IF102)
func runSwap(ctx context.Context, pool *pgxpool.Pool, client *http.Client, baseURL string, accounts []testAccount) {
	var id101, id102 int
	_ = pool.QueryRow(ctx, `SELECT id FROM courses WHERE code = 'IF101'`).Scan(&id101)
	_ = pool.QueryRow(ctx, `SELECT id FROM courses WHERE code = 'IF102'`).Scan(&id102)

	// Siapkan kondisi awal: 20 mahasiswa di IF101, 20 mahasiswa di IF102
	_, _ = pool.Exec(ctx, `DELETE FROM enrollments WHERE course_id IN ($1, $2)`, id101, id102)
	for i := 0; i < 20; i++ {
		_, _ = pool.Exec(ctx, `INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2)`, id101, accounts[i].User.ID)
	}
	for i := 20; i < 40; i++ {
		_, _ = pool.Exec(ctx, `INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2)`, id102, accounts[i].User.ID)
	}
	_, _ = pool.Exec(ctx, `UPDATE courses SET enrolled_count = 20 WHERE id IN ($1, $2)`, id101, id102)

	var deadlocksBefore int
	_ = pool.QueryRow(ctx, `SELECT deadlocks FROM pg_stat_database WHERE datname = current_database()`).Scan(&deadlocksBefore)

	fmt.Println("20 pasang mahasiswa pindah kelas berlawanan arah secara serentak...")
	url := fmt.Sprintf("%s/enrollments/swap", baseURL)

	start := time.Now()
	tally := fire(40, func(i int) string {
		var body map[string]int
		if i < 20 {
			body = map[string]int{"from_course_id": id101, "to_course_id": id102}
		} else {
			body = map[string]int{"from_course_id": id102, "to_course_id": id101}
		}
		return doRequest(client, http.MethodPost, url, accounts[i].Token, body)
	})
	elapsed := time.Since(start)

	var deadlocksAfter int
	_ = pool.QueryRow(ctx, `SELECT deadlocks FROM pg_stat_database WHERE datname = current_database()`).Scan(&deadlocksAfter)

	printTally(tally, elapsed)
	fmt.Printf("deadlock terdeteksi PostgreSQL selama uji: %d\n", deadlocksAfter-deadlocksBefore)
	checkCourseConsistency(ctx, pool, "IF101")
	checkCourseConsistency(ctx, pool, "IF102")
}

// Mode 4 (Tugas Mandiri D.3): 30 penetapan serentak ke 1 pembimbing berkuota 3
func runSupervisor(ctx context.Context, pool *pgxpool.Pool, client *http.Client, baseURL string, accounts []testAccount) {
	var supID int
	err := pool.QueryRow(ctx, `SELECT id FROM supervisors WHERE nidn = '0005089003'`).Scan(&supID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dosen pembimbing uji tidak ditemukan: %v\n", err)
		return
	}

	// Reset data supervisions sebelum pengujian dimulai
	_, _ = pool.Exec(ctx, `DELETE FROM supervisions`)
	_, _ = pool.Exec(ctx, `UPDATE supervisors SET supervised_count = 0`)

	// Ambil 30 ID mahasiswa dari tabel students
	rows, err := pool.Query(ctx, `SELECT id FROM students ORDER BY id LIMIT 30`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal mengambil 30 students: %v\n", err)
		return
	}
	defer rows.Close()

	var studentIDs []int
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		studentIDs = append(studentIDs, id)
	}
	if len(studentIDs) < 30 {
		fmt.Fprintf(os.Stderr, "jumlah baris di tabel students kurang dari 30 (ditemukan %d)\n", len(studentIDs))
		return
	}

	fmt.Println("30 mahasiswa berebut dosen pembimbing (kuota 3) secara serentak...")
	url := fmt.Sprintf("%s/supervisors/%d/assign", baseURL, supID)

	start := time.Now()
	tally := fire(30, func(i int) string {
		body := map[string]int{"student_id": studentIDs[i]}
		return doRequest(client, http.MethodPost, url, accounts[i].Token, body)
	})
	elapsed := time.Since(start)

	printTally(tally, elapsed)

	var quota, count, actual int
	_ = pool.QueryRow(ctx, `SELECT quota, supervised_count FROM supervisors WHERE id = $1`, supID).Scan(&quota, &count)
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM supervisions WHERE supervisor_id = $1`, supID).Scan(&actual)

	status := "KONSISTEN"
	if count != actual || count > quota || actual > quota {
		status = "TIDAK KONSISTEN"
	}
	fmt.Printf("Supervisor ID=%d kuota=%d supervised_count=%d baris supervisions=%d -> %s\n",
		supID, quota, count, actual, status)
}
