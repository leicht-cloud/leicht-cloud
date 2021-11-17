package memory

import (
	"bytes"
	"errors"
	"fmt"
)

type File struct {
	Data []byte

	filename string
	provider *StorageProvider
	read     *bytes.Reader
	write    *bytes.Buffer
}

func (f *File) Read(p []byte) (int, error) {
	if f.write != nil {
		return 0, errors.New("File was already opened in write mode")
	}
	if f.read == nil {
		if _, ok := f.provider.Data[f.filename]; !ok {
			return 0, fmt.Errorf("File doesn't exist yet, so can't be read: %s", f.filename)
		}
		f.read = bytes.NewReader(f.Data)
	}

	return f.read.Read(p)
}

func (f *File) Close() error {
	if f.write != nil {
		f.provider.Data[f.filename] = f.write.Bytes()
	}
	return nil
}

func (f *File) Write(p []byte) (int, error) {
	if f.read != nil {
		return 0, errors.New("File is already opened in read mode")
	}
	if f.write == nil {
		f.write = bytes.NewBuffer(f.Data)
	}

	return f.write.Write(p)
}
