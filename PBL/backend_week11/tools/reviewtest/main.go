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
	"latihan-fiber/helper"
	"latihan-fiber/app/model"
)

const baseURL = "http://localhost:3000/api/v1"

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

	// Bersihkan ulasan kelas 1 di MongoDB dan siapkan pendaftaran KRS di PostgreSQL
	_, _ = mongoDB.Collection("course_reviews").DeleteMany(ctx, bson.M{"course_id": 1})
	_, _ = pool.Exec(ctx, `DELETE FROM enrollments WHERE course_id = 1`)

	// Pastikan user admin (ID 1), mhs001, mhs002, mhs003 tersedia
	ensureUser := func(username, email, role string) int {
		var id int
		err := pool.QueryRow(ctx, `
			INSERT INTO users (username, email, password_hash, full_name, role)
			VALUES ($1, $2, 'hash', $1, $3)
			ON CONFLICT (username) DO UPDATE SET role = EXCLUDED.role
			RETURNING id`, username, email, role).Scan(&id)
		if err != nil {
			_ = pool.QueryRow(ctx, `SELECT id FROM users WHERE username = $1`, username).Scan(&id)
		}
		return id
	}

	adminID := ensureUser("admin_review", "admin_rev@kampus.ac.id", "admin")
	mhs1ID := ensureUser("mhs001", "mhs001@kampus.ac.id", "user")
	mhs2ID := ensureUser("mhs002", "mhs002@kampus.ac.id", "user")
	mhs3ID := ensureUser("mhs003", "mhs003@kampus.ac.id", "user")

	// Daftarkan mhs001 dan mhs002 ke kelas IF101 (course_id = 1), mhs003 TIDAK terdaftar
	_, _ = pool.Exec(ctx, `INSERT INTO enrollments (course_id, user_id) VALUES (1, $1), (1, $2) ON CONFLICT DO NOTHING`, mhs1ID, mhs2ID)
	_, _ = pool.Exec(ctx, `UPDATE courses SET enrolled_count = (SELECT COUNT(*) FROM enrollments WHERE course_id = 1) WHERE id = 1`)

	jwtMgr := helper.NewJWTManager(config.GetEnv("JWT_SECRET", ""), config.GetEnv("JWT_ISSUER", "praktikum-backend"), 3600*1e9)
	adminUser := model.User{ID: adminID, Username: "admin_review", Role: "admin"}
	mhs1User := model.User{ID: mhs1ID, Username: "mhs001", Role: "user"}
	mhs2User := model.User{ID: mhs2ID, Username: "mhs002", Role: "user"}
	mhs3User := model.User{ID: mhs3ID, Username: "mhs003", Role: "user"}
	tokAdmin, _ := jwtMgr.GenerateAccess(adminUser)
	tokMhs1, _ := jwtMgr.GenerateAccess(mhs1User)
	tokMhs2, _ := jwtMgr.GenerateAccess(mhs2User)
	tokMhs3, _ := jwtMgr.GenerateAccess(mhs3User)

	fmt.Println("=== PENGUJIAN ENDPOINT ULASAN KELAS (LANGKAH 9) ===")

	// 1. mhs003 (tidak terdaftar) menulis ulasan IF101 -> 403 NOT_ENROLLED
	st, body, _ := req("POST", "/courses/1/reviews", tokMhs3, `{"rating":5,"comment":"Saya sebenarnya belum ambil kelas ini","tags":["seru"]}`)
	fmt.Printf("1. mhs003 menulis ulasan IF101           -> %d | %s\n", st, compact(body))

	// 2. mhs001 menulis ulasan dengan tag berantakan -> 201
	st, body, loc := req("POST", "/courses/1/reviews", tokMhs1, `{"rating":5,"comment":"Materi Go-nya jelas, tugasnya cukup banyak","tags":["Tugas-Banyak"," dosen-jelas ","tugas-banyak"]}`)
	review1ID := extractID(body)
	fmt.Printf("2. mhs001 menulis ulasan IF101           -> %d | Location: %s | ID: %s\n", st, loc, review1ID)

	// 3. mhs001 menulis ulasan kedua di kelas yang sama -> 409 ALREADY_REVIEWED
	st, body, _ = req("POST", "/courses/1/reviews", tokMhs1, `{"rating":4,"comment":"Mencoba kirim ulasan kedua kalinya","tags":["seru"]}`)
	fmt.Printf("3. mhs001 menulis ulasan kedua           -> %d | %s\n", st, compact(body))

	// 4. Rating 9, komentar "pendek", 6 tag -> 422
	st, body, _ = req("POST", "/courses/1/reviews", tokMhs2, `{"rating":9,"comment":"pendek","tags":["t1","t2","t3","t4","t5","t6"]}`)
	fmt.Printf("4. Validasi gagal (rating 9, 6 tag)      -> %d | %s\n", st, compact(body))

	// 5. Ulasan untuk kelas 99999 -> 404
	st, body, _ = req("POST", "/courses/99999/reviews", tokMhs1, `{"rating":5,"comment":"Kelas ini tidak ada di database","tags":["seru"]}`)
	fmt.Printf("5. Ulasan untuk kelas 99999              -> %d | %s\n", st, compact(body))

	// 6. GET /reviews/bukan-objectid -> 400
	st, body, _ = req("GET", "/reviews/bukan-objectid", tokMhs1, "")
	fmt.Printf("6. GET /reviews/bukan-objectid           -> %d | %s\n", st, compact(body))

	// 7. mhs002 mengubah ulasan mhs001 -> 403
	st, body, _ = req("PATCH", "/reviews/"+review1ID, tokMhs2, `{"rating":1}`)
	fmt.Printf("7. mhs002 mengubah ulasan mhs001         -> %d | %s\n", st, compact(body))

	// 8. mhs001 mengubah ulasannya sendiri -> 200
	st, body, _ = req("PATCH", "/reviews/"+review1ID, tokMhs1, `{"rating":4,"comment":"Materi Go-nya sangat jelas dan terstruktur"}`)
	fmt.Printf("8. mhs001 mengubah ulasannya sendiri     -> %d | %s\n", st, compact(body))

	// 9. mhs001 menilai ulasannya sendiri "membantu" -> 409 OWN_REVIEW
	st, body, _ = req("POST", "/reviews/"+review1ID+"/helpful", tokMhs1, `{}`)
	fmt.Printf("9. mhs001 menilai ulasan sendiri         -> %d | %s\n", st, compact(body))

	// 10. 1 mahasiswa (mhs002) menilai 20 kali serentak -> 1x 200, 19x 409 ALREADY_VOTED
	var ok200, conflict409 int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, _, _ := req("POST", "/reviews/"+review1ID+"/helpful", tokMhs2, `{}`)
			if s == 200 {
				atomic.AddInt32(&ok200, 1)
			} else if s == 409 {
				atomic.AddInt32(&conflict409, 1)
			}
		}()
	}
	wg.Wait()
	fmt.Printf("10. mhs002 menilai 20x serentak          -> 200 OK: %d, 409 ALREADY_VOTED: %d\n", ok200, conflict409)

	// 11. mhs002 membuat ulasan untuk diuji moderasi hapus
	_, body2, _ := req("POST", "/courses/1/reviews", tokMhs2, `{"rating":5,"comment":"Ulasan dari mhs002 untuk kelas IF101","tags":["praktikal","dosen-jelas"]}`)
	review2ID := extractID(body2)

	// 12. mhs003 menghapus ulasan mhs002 -> 403
	st, body, _ = req("DELETE", "/reviews/"+review2ID, tokMhs3, "")
	fmt.Printf("11. mhs003 menghapus ulasan mhs002       -> %d | %s\n", st, compact(body))

	// 13. admin menghapus ulasan mhs002 -> 204
	st, _, _ = req("DELETE", "/reviews/"+review2ID, tokAdmin, "")
	fmt.Printf("12. admin menghapus ulasan mhs002        -> %d (No Content)\n", st)

	// 14. Menghapus ulasan yang sudah terhapus -> 404
	st, body, _ = req("DELETE", "/reviews/"+review2ID, tokAdmin, "")
	fmt.Printf("13. Hapus ulasan yang sudah terhapus     -> %d | %s\n", st, compact(body))

	// 15. NoSQL Injection di body {"comment": {"$ne": ""}} -> 400
	st, body, _ = req("POST", "/courses/1/reviews", tokMhs2, `{"rating":5,"comment":{"$ne":""}}`)
	fmt.Printf("14. NoSQL Injection di body JSON         -> %d | %s\n", st, compact(body))

	// 16. ?tag={"$ne":""} pada daftar ulasan -> 200 dengan 0 hasil
	st, body, _ = req("GET", `/courses/1/reviews?tag=%7B%22$ne%22:%22%22%7D`, tokMhs1, "")
	fmt.Printf("15. ?tag={\"$ne\":\"\"} pada daftar ulasan  -> %d | %s\n", st, compact(body))

	// 17. Ringkasan ulasan kelas (/courses/1/reviews/summary) -> 200
	st, body, _ = req("GET", "/courses/1/reviews/summary", tokMhs1, "")
	fmt.Printf("16. GET /courses/1/reviews/summary       -> %d | %s\n", st, compact(body))
}

func req(method, path, token, jsonBody string) (int, string, string) {
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
		return 0, err.Error(), ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), resp.Header.Get("Location")
}

func extractID(raw string) string {
	var parsed struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &parsed)
	return parsed.Data.ID
}

func compact(s string) string {
	if len(s) > 110 {
		return s[:110] + "..."
	}
	return s
}