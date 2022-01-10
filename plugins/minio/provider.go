package main

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type StorageProvider struct {
	Client *minio.Client

	Hostname        string `yaml:"hostname"`
	Https           bool   `yaml:"https"`
	AccessKey       string `yaml:"access_key"`
	SecretAccessKey string `yaml:"secret_key"`
	Prefix          string `yaml:"prefix"`
	Region          string `yaml:"region"`
}

func (s *StorageProvider) OnConfigure() error {
	client, err := minio.New(s.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretAccessKey, ""),
		Secure: s.Https,
	})
	if err != nil {
		return err
	}

	s.Client = client
	s.Client.SetAppInfo("Go-Cloud", "0.0")
	return nil
}

func (s *StorageProvider) bucketName(user *models.User) string {
	return fmt.Sprintf("%s%d", s.Prefix, user.ID)
}

func (s *StorageProvider) InitUser(ctx context.Context, user *models.User) error {
	err := s.Client.MakeBucket(ctx, s.bucketName(user), minio.MakeBucketOptions{
		Region: s.Region,
	})

	if resp, ok := err.(minio.ErrorResponse); ok {
		// TODO: There is probably a better way to detect if the bucket we tried to create already exists?
		if resp.Message == "Your previous request to create the named bucket succeeded and you already own it." {
			return nil
		}
	}

	return err
}

const dir_file = "This file is just here to create the folder, as minio won't let you have empty folders."

func (s *StorageProvider) Mkdir(ctx context.Context, user *models.User, dir string) error {
	// as minio doesn't seem to have the concept of creating a folder and rather creates them on the fly as needed
	// we can't really implement a mkdir call. So instead we just create a tiny file inside the folder
	// we want to create. The contents of this file will be static and will just explain why this file is there.
	// it is important to note that we ignore this file in the ListDirectory call, so users of go-cloud
	// will never actually see this file. The explanation is just there for the admin.
	_, err := s.Client.PutObject(ctx,
		s.bucketName(user),
		path.Join(dir, ".go-cloud"),
		strings.NewReader(dir_file),
		int64(len(dir_file)), minio.PutObjectOptions{
			ContentType: "text/plain",
		})
	return err
}

// TODO: Currently cannot move directories
func (s *StorageProvider) Move(ctx context.Context, user *models.User, src, dst string) error {
	// there is no direct moving in minio, so instead we make a copy and then delete the original
	_, err := s.Client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: s.bucketName(user),
		Object: dst,
	}, minio.CopySrcOptions{
		Bucket: s.bucketName(user),
		Object: src,
	})
	if err != nil {
		return err
	}
	return s.Delete(ctx, user, src)
}

func (s *StorageProvider) minioObjToFileInfo(obj minio.ObjectInfo, dir string) (*storage.FileInfo, error) {
	if obj.Err != nil {
		return nil, obj.Err
	}

	isDir := strings.HasSuffix(obj.Key, "/")
	path := strings.TrimSuffix(obj.Key, "/")

	return &storage.FileInfo{
		Name:      strings.TrimPrefix(path, dir),
		FullPath:  path,
		UpdatedAt: obj.LastModified,
		Size:      uint64(obj.Size),
		Directory: isDir,
	}, nil
}

func (s *StorageProvider) ListDirectory(ctx context.Context, user *models.User, dir string) (<-chan storage.FileInfo, error) {
	if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}
	logrus.Debugf("%+v.ListDirectory(%s)", s, dir)
	ch := s.Client.ListObjects(ctx, s.bucketName(user), minio.ListObjectsOptions{
		Recursive: false,
		Prefix:    dir,
	})

	out := make(chan storage.FileInfo, 1)

	fi, err := s.minioObjToFileInfo(<-ch, dir)
	if err != nil {
		return nil, err
	}
	out <- *fi

	go func(out chan<- storage.FileInfo, ch <-chan minio.ObjectInfo) {
		for entry := range ch {
			// this is the holder file created by mkdir, so we ignore it
			if strings.HasSuffix(entry.Key, ".go-cloud") {
				continue
			} else if entry.Key == "/" {
				continue
			}

			fi, err = s.minioObjToFileInfo(entry, dir)
			if err == nil {
				out <- *fi
			}
		}

		close(out)
	}(out, ch)

	return out, nil
}

func (s *StorageProvider) File(ctx context.Context, user *models.User, fullpath string) (storage.File, error) {
	logrus.Debugf("Requesting file: %s", fullpath)
	return &File{
		Fullpath: fullpath,
		Bucket:   s.bucketName(user),
		Client:   s.Client,
	}, nil
}

func (s *StorageProvider) Delete(ctx context.Context, user *models.User, fullpath string) error {
	return s.Client.RemoveObject(ctx, s.bucketName(user), fullpath, minio.RemoveObjectOptions{})
}
