package main

import "fmt"

func swap(a, b *int) {
	*a, *b = *b, *a //membongkar isi dari kedua alamat memori tersebut, lalu saling menyilangkan nilainya secara permanen
}

//menambah isi slice dengan pointer
func updateSlice(s *[]string, newItem string) { //agar memastikan slice asli tertimpa dengan data yang bertambah, kita mengirimkan alamat memorinya (*[]string).
	*s = append(*s, newItem) //membongkar isi dari alamat memori slice, lalu menambahkan data baru ke dalamnya
}

func ubahDenganValue(nilai int) { //pass by value, nilai asli tidak berubah
	nilai = 100
}

func ubahDenganPointer(nilai *int) { //pass by pointer, nilai asli berubah karena kita mengirimkan alamat memorinya (*int)
	*nilai = 100
}

func main() {
	angka1 := 10
	angka2 := 20

	fmt.Println("Swap dengan Pointer")
	fmt.Println("Sebelum ditukar:")
	fmt.Println("Angka 1:", angka1)
	fmt.Println("Angka 2:", angka2)

	swap(&angka1, &angka2)

	fmt.Println("\nSetelah ditukar:")
	fmt.Println("Angka 1:", angka1)
	fmt.Println("Angka 2:", angka2)

	hobi := []string{"Coding", "Mukbang"}

	fmt.Println("\nUpdate Slice")
	fmt.Println("Sebelum ditambahkan:", hobi)

	updateSlice(&hobi, "Streaming")

	fmt.Println("Setelah ditambahkan:", hobi)

	nilaiValue := 50

	fmt.Println("\nPass by Value")
	fmt.Println("Sebelum function:", nilaiValue)

	ubahDenganValue(nilaiValue)

	fmt.Println("Setelah function:", nilaiValue)

	nilaiPointer := 50

	fmt.Println("\nPass by Pointer")
	fmt.Println("Sebelum function:", nilaiPointer)

	ubahDenganPointer(&nilaiPointer)

	fmt.Println("Setelah function:", nilaiPointer)
}