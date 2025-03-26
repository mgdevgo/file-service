package file

type Page struct {
	Number int
	Size   int
}

func NewPage(number int, size int) Page {
	if number < 1 {
		number = 1
	}
	if size < 1 {
		size = 10
	}
	return Page{
		Number: number,
		Size:   size,
	}
}
