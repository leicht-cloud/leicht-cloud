package plugin

import (
	"bytes"
	"context"
	"errors"
	"io"
)

type File struct {
	Storage *GrpcStorage
	Id      int32

	readbuf bytes.Buffer
	reader  StorageProvider_ReadFileClient
	isEOF   bool
	writing bool
}

func (f *File) Read(p []byte) (int, error) {
	if f.writing {
		return 0, errors.New("File is already opened in write mode")
	}

	if f.isEOF {
		return f.readTilEOF(p)
	}

	if f.reader == nil {
		reader, err := f.Storage.Client.ReadFile(context.Background(),
			&ReadFileQuery{
				Id: f.Id,
			},
		)
		if err != nil {
			return 0, err
		}
		f.reader = reader
	}

	// while our buffer doesn't have enough data left we read from the stream first
	for f.readbuf.Len() < len(p) {
		reply, err := f.reader.Recv()
		if err != nil {
			return 0, err
		}
		if reply.GetError() != nil && reply.GetError().GetMessage() != "" {
			return 0, errors.New(reply.GetError().GetMessage())
		}

		_, err = f.readbuf.Write(reply.Data)
		if err != nil {
			return 0, err
		}

		if reply.GetEOF() {
			f.isEOF = true
			return f.readTilEOF(p)
		}
	}

	return f.readbuf.Read(p)
}

// internal use only, ASSUMES isEOF is true
func (f *File) readTilEOF(p []byte) (int, error) {
	if f.readbuf.Len() > 0 {
		n, err := f.readbuf.Read(p)
		if f.readbuf.Len() == 0 {
			return n, io.EOF
		}
		return n, err
	}
	return 0, io.EOF
}

func (f *File) Close() (err error) {
	if f.reader != nil {
		err = f.reader.CloseSend()
	}
	err2 := f.Storage.closeFile(f.Id)
	if err != nil {
		return err
	} else if err2 != nil {
		return err2
	}
	return nil
}

func (f *File) Write(p []byte) (int, error) {
	if f.reader != nil {
		return 0, errors.New("File is already opened in read mode")
	}

	f.writing = true

	query := &WriteFileQuery{
		Id:   f.Id,
		Data: p,
	}

	reply, err := f.Storage.Client.WriteFile(context.TODO(), query)
	if err != nil {
		return 0, err
	}
	if reply.GetError().GetMessage() != "" {
		return 0, errors.New(reply.GetError().GetMessage())
	}

	return int(reply.SizeWritten), nil
}
