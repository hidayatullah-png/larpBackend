package main

import "fmt"

func main() {
	nama := "Leon"

	umur := 21

	ipk := 3.75

	mahasiswaAktif := true

	hobi := []string{"Coding", "Mukbang", "Streaming"}

	fmt.Println("Data Variabel ")
	fmt.Println("Nama:", nama)
	fmt.Println("Umur:", umur)
	fmt.Println("IPK:", ipk)
	fmt.Println("Mahasiswa Aktif:", mahasiswaAktif)
	fmt.Println("Hobi:", hobi)

	mahasiswa := make(map[string]int)

	mahasiswa["Krauser"] = 90
	mahasiswa["Wesker"] = 85
	mahasiswa["Chris"] = 88

	fmt.Println("\nData Mahasiswa")
	fmt.Println(mahasiswa)

	nilai, ada := mahasiswa["Krauser"]

	if ada {
		fmt.Println("\nNilai Krauser:", nilai)
	} else {
		fmt.Println("\nData Krauser tidak ditemukan")
	}

	nilai, ada = mahasiswa["Carlos"]

	if ada {
		fmt.Println("Nilai Carlos:", nilai)
	} else {
		fmt.Println("Data Carlos tidak ditemukan ")
	}

	delete(mahasiswa, "Wesker")

	fmt.Println("\nSetelah Data Wesker Dihapus :")
	fmt.Println(mahasiswa)

	fmt.Println("\nSeluruh Data Mahasiswa ")

	for nama, nilai := range mahasiswa {
		fmt.Println("Nama:", nama, ", Nilai:", nilai)
	}
}
