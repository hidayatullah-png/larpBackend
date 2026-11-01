package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/v2/bson"
	"latihan-fiber/config"
	"latihan-fiber/database"
)

const baseURL = "http://localhost:3000/api/v1"

// loginOrRegister mendapatkan access_token resmi langsung dari endpoint /auth/login server
func loginOrRegister(ctx context.Context, pool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (any, error)
}, username, email, role string) (int, string) {
	password := "Praktikum#2026"
	// Coba register dulu (abaikan jika sudah terdaftar)
	regBody := fmt.Sprintf(`{"username":"%s","email":"%s","password":"%s","full_name":"%s"}`, username, email, password, username)
	req("POST", "/auth/register", "", regBody)

	// Login untuk mendapatkan access_token asli dari server
	_, body := req("POST", "/auth/login", "", fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password))
	var parsed struct {
		Data struct {
			AccessToken string `json:"access_token"`
			Token       string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(body), &parsed)
	tok := parsed.Data.AccessToken
	if tok == "" {
		tok = parsed.Data.Token
	}
	return 0, tok
}

func main() {
	config.LoadEnv()
	ctx := context.Background()

	pool, err := database.NewPool(ctx)
	if err != nil {
		fmt.Println("gagal konek postgres:", err)
		os.Exit(1)
	}
	defer pool.Close()

	mongoClient, err := database.NewMongoClient(ctx)
	if err != nil {
		fmt.Println("gagal konek mongo:", err)
		os.Exit(1)
	}
	defer mongoClient.Disconnect(ctx)
	mongoDB := mongoClient.Database(config.GetEnv("MONGO_DATABASE", "praktikum_backend"))

	// Buat / login akun penguji resmi lewat API /auth/register & /auth/login
	password := "Praktikum#2026"
	_, _ = req("POST", "/auth/register", "", fmt.Sprintf(`{"username":"tester_pf","email":"tester_pf@kampus.ac.id","password":"%s","full_name":"Tester Portfolio"}`, password))
	_, _ = pool.Exec(ctx, `UPDATE users SET role = 'admin' WHERE username = 'tester_pf'`)

	_, loginResp := req("POST", "/auth/login", "", fmt.Sprintf(`{"username":"tester_pf","password":"%s"}`, password))
	token := extractToken(loginResp)
	if token == "" {
		fmt.Println("Gagal mendapatkan token dari /auth/login. Response:", loginResp)
		os.Exit(1)
	}

	var userID int
	_ = pool.QueryRow(ctx, `SELECT id FROM users WHERE username = 'tester_pf'`).Scan(&userID)

	// Pastikan student ID tersedia di PostgreSQL
	var studentID int
	err = pool.QueryRow(ctx, `SELECT id FROM students ORDER BY id LIMIT 1`).Scan(&studentID)
	if err != nil {
		_ = pool.QueryRow(ctx, `INSERT INTO students (nim, name, grade, owner_id) VALUES ('434241087', 'Hidayatullah Sukma Dewi', 3.90, $1) RETURNING id`, userID).Scan(&studentID)
	}

	// Bersihkan portofolio student ini sebelum diuji
	_, _ = mongoDB.Collection("student_portfolios").DeleteMany(ctx, bson.M{"student_id": studentID})

	stuPath := fmt.Sprintf("/students/%d/portfolio", studentID)

	fmt.Println("=== BUKTI C.3.1: PENGUJIAN ENDPOINT PORTOFOLIO & 3 KASUS GAGAL ===")

	// 1. PUT /students/:id/portfolio (Sukses 200)
	st, body := req("PUT", stuPath, token, `{
		"headline": "Backend & Security Engineer | Go, PostgreSQL & MongoDB",
		"bio": "Mahasiswa D4 Teknik Informatika Universitas Airlangga dengan fokus pengembangan API dan web security.",
		"certificates": [{"name": "Junior Penetration Tester", "issuer": "CyberSecurity Academy", "date": "2026-08-15"}]
	}`)
	fmt.Printf("1. PUT %s                      -> %d | %s\n", stuPath, st, compact(body))

	// 2. POST /students/:id/portfolio/projects (Sukses 201)
	st, body = req("POST", stuPath+"/projects", token, `{
		"title": "FELIPE - Feline Expert System",
		"technologies": ["Go", "Fiber", "PostgreSQL", "Clean-Architecture"],
		"link": "https://github.com/sukma/felipe-expert-system",
		"year": 2026
	}`)
	proj1ID := extractLastProjectID(body)
	fmt.Printf("2. POST %s/projects (Proyek 1) -> %d | ProjectID: %s\n", stuPath, st, proj1ID)

	// Tambah proyek ke-2 untuk diuji DELETE ($pull)
	_, body2 := req("POST", stuPath+"/projects", token, `{
		"title": "Proyek Sementara",
		"technologies": ["PHP", "Laravel"],
		"link": "https://github.com/sukma/temp",
		"year": 2025
	}`)
	tempProjID := extractLastProjectID(body2)

	// 3. DELETE /students/:id/portfolio/projects/:projectId (Sukses 200)
	st, body = req("DELETE", stuPath+"/projects/"+tempProjID, token, "")
	fmt.Printf("3. DELETE %s/projects/%s -> %d | %s\n", stuPath, tempProjID, st, compact(body))

	// 4. GET /students/:id/portfolio (Sukses 200)
	st, body = req("GET", stuPath, token, "")
	fmt.Printf("4. GET %s                      -> %d | %s\n", stuPath, st, compact(body))

	// 5. Kasus Gagal 1: Mahasiswa tidak ada di PostgreSQL (99999 -> 404)
	st, body = req("PUT", "/students/99999/portfolio", token, `{
		"headline": "Headline Mahasiswa Fiktif",
		"bio": "Bio mahasiswa yang tidak ada di PostgreSQL."
	}`)
	fmt.Printf("5. [GAGAL 1 - Mahasiswa 99999 Tidak Ada]      -> %d | %s\n", st, compact(body))

	// 6. Kasus Gagal 2: Validasi gagal (year: 1990, technologies kosong -> 422)
	st, body = req("POST", stuPath+"/projects", token, `{
		"title": "AB",
		"technologies": [],
		"link": "bukan-url",
		"year": 1990
	}`)
	fmt.Printf("6. [GAGAL 2 - Validasi Input Gagal 422]       -> %d | %s\n", st, compact(body))

	// 7. Kasus Gagal 3: Format ObjectID proyek tidak valid pada DELETE (-> 400)
	st, body = req("DELETE", stuPath+"/projects/bukan-object-id", token, "")
	fmt.Printf("7. [GAGAL 3 - ObjectID Proyek Tidak Valid]    -> %d | %s\n", st, compact(body))

	fmt.Println("\n=== BUKTI C.3.3: BUKTI NOSQL INJECTION DITOLAK ===")
	st, body = req("PUT", stuPath, token, `{"headline": {"$ne": ""}, "bio": "Percobaan injeksi operator NoSQL"}`)
	fmt.Printf("8. [NoSQL Injection di Body JSON]             -> %d | %s\n", st, compact(body))

	fmt.Println("\n=== BUKTI C.3.2: UJI SERENTAK BATAS 20 PROYEK (18 AWAL + 10 SERENTAK) ===")
	techPool := [][]string{
		{"go", "fiber", "mongodb"},
		{"go", "postgresql", "jwt"},
		{"python", "scikit-learn"},
		{"php", "laravel", "mysql"},
		{"renpy", "python"},
	}
	for i := 2; i <= 18; i++ {
		tList, _ := json.Marshal(techPool[i%len(techPool)])
		payload := fmt.Sprintf(`{"title":"Proyek Portofolio #%d","technologies":%s,"link":"https://github.com/sukma/proj-%d","year":2026}`, i, string(tList), i)
		req("POST", stuPath+"/projects", token, payload)
	}
	fmt.Println("Persiapan selesai: portofolio kini berisi 18 proyek.")
	fmt.Println("Menembakkan 10 request POST /projects secara serentak...")

	var created201, limit409 int32
	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			payload := fmt.Sprintf(`{"title":"Proyek Serentak #%d","technologies":["go","mongodb","docker"],"link":"https://github.com/sukma/concurrent-%d","year":2026}`, n, n)
			s, _ := req("POST", stuPath+"/projects", token, payload)
			if s == 201 {
				atomic.AddInt32(&created201, 1)
			} else if s == 409 {
				atomic.AddInt32(&limit409, 1)
			}
		}(i)
	}
	wg.Wait()

	_, finalPortfolio := req("GET", stuPath, token, "")
	finalCount := countProjects(finalPortfolio)
	fmt.Printf("Hasil 10 request serentak -> 201 Created: %d | 409 PROJECT_LIMIT: %d | Jumlah proyek akhir: %d\n",
		created201, limit409, finalCount)

	fmt.Println("\n=== BUKTI C.2.4: GET /portfolios/stats/technologies (2x $unwind) ===")
	st, body = req("GET", "/portfolios/stats/technologies", token, "")
	fmt.Printf("GET /portfolios/stats/technologies -> %d\n%s\n", st, body)
}

func extractToken(raw string) string {
	var parsed struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &parsed)
	for _, k := range []string{"access_token", "token", "accessToken"} {
		if v, ok := parsed.Data[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func req(method, path, token, jsonBody string) (int, string) {
	var bodyReader io.Reader
	if jsonBody != "" {
		bodyReader = bytes.NewBufferString(jsonBody)
	}
	r, _ := http.NewRequest(method, baseURL+path, bodyReader)
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func extractLastProjectID(raw string) string {
	var parsed struct {
		Data struct {
			Projects []struct {
				ID string `json:"id"`
			} `json:"projects"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &parsed)
	if len(parsed.Data.Projects) == 0 {
		return ""
	}
	return parsed.Data.Projects[len(parsed.Data.Projects)-1].ID
}

func countProjects(raw string) int {
	var parsed struct {
		Data struct {
			Projects []any `json:"projects"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &parsed)
	return len(parsed.Data.Projects)
}

func compact(s string) string {
	if len(s) > 105 {
		return s[:105] + "..."
	}
	return s
}