package main

type Writer interface {
	Write([]byte) (float64, error)
}

type FileWriter struct {
	filename string
}

func (f *FileWriter) Write(data []byte) (float64, error) {
	// implementation
	return float64(len(data)), nil
}
