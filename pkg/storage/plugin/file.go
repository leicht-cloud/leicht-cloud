package plugin

import (
	"bytes"
	"context"
	"errors"
)

type File struct {
	Storage *GrpcStorage
	Id      uint64

	buffer bytes.Buffer
	reader StorageProvider_ReadFileClient
	writer StorageProvider_WriteFileClient
}

func (f *File) Read(p []byte) (int, error) {
	if f.writer != nil {
		return 0, errors.New("File is already opened in write mode")
	}

	if f.reader == nil {
		reader, err := f.Storage.Client.ReadFile(context.Background(),
			&ReadFileQuery{
				Id: f.Id,
			},
		)
		if err != nil {
			return -1, err
		}
		f.reader = reader
	}

	// while our buffer doesn't have enough data left we read from the stream first
	for f.buffer.Len() < len(p) {
		reply, err := f.reader.Recv()
		if err != nil {
			return -1, err
		}
		_, err = f.buffer.Write(reply.Data)
		if err != nil {
			return -1, err
		}
	}

	return f.buffer.Read(p)
}

func (f *File) Close() error {
	defer f.Storage.closeFile(f.Id)
	if f.reader != nil {
		return f.reader.CloseSend()
	}
	if f.writer != nil {
		return f.writer.CloseSend()
	}
	return nil
}

func (f *File) Write(p []byte) (int, error) {
	if f.reader != nil {
		return 0, errors.New("File is already opened in read mode")
	}

	if f.writer == nil {
		writer, err := f.Storage.Client.WriteFile(context.Background())
		if err != nil {
			return -1, err
		}

		f.writer = writer
	}

	err := f.writer.Send(
		&WriteFileQuery{
			Id:   f.Id,
			Data: p,
		},
	)
	if err != nil {
		return -1, err
	}
	return len(p), nil
}
