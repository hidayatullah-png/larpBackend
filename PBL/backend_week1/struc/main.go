package main

import "fmt"

type Student struct { //struct (tipe data bentukan yang menggabungkan beberapa variabel dengan tipe data berbeda ke dalam satu nama)
	ID       int
	Name     string
	Grade    float64
	IsActive bool
}

func (s Student) GetInfo() string { //method (fungsi yang terkait dengan struct)
	return fmt.Sprintf(
		"ID: %d | Nama: %s | Nilai: %.2f | Aktif: %t",
		s.ID, s.Name, s.Grade, s.IsActive,
	)
}

func (s *Student) UpdateGrade(grade float64) { //method dengan pointer receiver (agar dapat mengubah nilai asli dari struct)
	s.Grade = grade
}

func (s *Student) Activate() {
	s.IsActive = true
}

func (s *Student) Deactivate() {
	s.IsActive = false
}

func main() {
	student := Student{
		ID:       101,
		Name:     "Leon",
		Grade:    85.5,
		IsActive: false,
	}

	fmt.Println("Data Student")
	fmt.Println(student.GetInfo())

	student.UpdateGrade(92.5)

	fmt.Println("\nSetelah Nilai Diubah")
	fmt.Println(student.GetInfo())

	student.Activate()

	fmt.Println("\nSetelah Student Diaktifkan")
	fmt.Println(student.GetInfo())

	student.Deactivate()

	fmt.Println("\nSetelah Student Dinonaktifkan")
	fmt.Println(student.GetInfo())
}
