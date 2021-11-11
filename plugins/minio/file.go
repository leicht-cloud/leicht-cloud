package main

import (
	"context"
	"io"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

type File struct {
	Fullpath string
	Bucket   string
	Client   *minio.Client

	read *minio.Object

	writer *io.PipeWriter
	wg     sync.WaitGroup
}

func (f *File) Read(p []byte) (int, error) {
	if f.read == nil {
		obj, err := f.Client.GetObject(context.TODO(), f.Bucket, f.Fullpath, minio.GetObjectOptions{})
		if err != nil {
			return 0, err
		}

		f.read = obj
	}

	return f.read.Read(p)
}

func (f *File) Close() (err error) {
	if f.read != nil {
		err = f.read.Close()
	} else if f.writer != nil {
		err = f.writer.Close()
	}
	f.wg.Wait()
	return err
}

func (f *File) Write(p []byte) (int, error) {
	if f.writer == nil {
		r, w := io.Pipe()
		f.writer = w

		f.wg.Add(1)
		go func(reader io.Reader) {
			logrus.Debug("Uploading in a separate goroutine..")
			upload, err := f.Client.PutObject(context.Background(), f.Bucket, f.Fullpath, r, -1, minio.PutObjectOptions{})
			logrus.Debug(upload, err)
			f.wg.Done()
		}(r)
	}

	return f.writer.Write(p)
}
