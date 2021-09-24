package local

import (
	"errors"
	"os"
)

type File struct {
	FullPath string

	read  *os.File
	write *os.File
}

func (f *File) Read(p []byte) (int, error) {
	if f.write != nil {
		return 0, errors.New("File is already opened in write mode")
	}
	if f.read == nil {
		file, err := os.OpenFile(f.FullPath, os.O_RDONLY, 0700)
		if err != nil {
			return 0, err
		}

		f.read = file
	}

	return f.read.Read(p)
}

func (f *File) Close() error {
	if f.read != nil {
		return f.read.Close()
	}
	if f.write != nil {
		return f.write.Close()
	}
	return nil
}

func (f *File) Write(p []byte) (int, error) {
	if f.read != nil {
		return 0, errors.New("File is already opened in read mode")
	}
	if f.write == nil {
		file, err := os.OpenFile(f.FullPath, os.O_WRONLY|os.O_CREATE, 0700)
		if err != nil {
			return 0, err
		}

		f.write = file
	}

	return f.write.Write(p)
}
