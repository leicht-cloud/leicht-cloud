package plugin

import (
	context "context"
	"errors"
	"time"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

type GrpcStorage struct {
	Conn   *grpc.ClientConn
	Client StorageProviderClient

	openFiles map[uint64]*File
}

func toError2(err *Error, Err error) error {
	if Err != nil {
		return Err
	}
	if err != nil || err.GetMessage() != "" {
		return errors.New(err.GetMessage())
	}
	return nil
}

func NewGrpcStorage(addr string) (*GrpcStorage, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	if err != nil {
		return nil, err
	}
	return &GrpcStorage{
		Conn:      conn,
		Client:    NewStorageProviderClient(conn),
		openFiles: make(map[uint64]*File),
	}, nil
}

func (s *GrpcStorage) Close() error {
	for _, f := range s.openFiles {
		f.Close()
	}
	return nil
}

func (s *GrpcStorage) InitUser(ctx context.Context, user *models.User) error {
	err, Err := s.Client.InitUser(ctx,
		&User{
			Id: user.ID,
		},
	)
	return toError2(err, Err)
}

func (s *GrpcStorage) Mkdir(ctx context.Context, user *models.User, path string) error {
	err, Err := s.Client.MkDir(ctx,
		&MkdirQuery{
			User: &User{
				Id: user.ID,
			},
			Path: path},
	)
	return toError2(err, Err)
}

func (s *GrpcStorage) Move(ctx context.Context, user *models.User, src, dst string) error {
	err, Err := s.Client.Move(ctx,
		&MoveQuery{
			User: &User{
				Id: user.ID,
			},
			Src: src,
			Dst: dst,
		},
	)
	return toError2(err, Err)
}

func (s *GrpcStorage) ListDirectory(ctx context.Context, user *models.User, path string) (*storage.DirectoryInfo, error) {
	dir, err := s.Client.ListDirectory(ctx,
		&ListDirectoryQuery{
			User: &User{
				Id: user.ID,
			},
			Path: path,
		},
	)
	outErr := toError2(dir.Error, err)
	if outErr != nil {
		return nil, err
	}

	out := &storage.DirectoryInfo{
		Path:  path,
		Files: make([]storage.FileInfo, 0, len(dir.Files)),
	}

	for _, file := range dir.Files {
		out.Files = append(out.Files, storage.FileInfo{
			Name:      file.Name,
			FullPath:  file.FullPath,
			MimeType:  file.MimeType,
			CreatedAt: time.Unix(int64(file.CreatedAt), 0),
			UpdatedAt: time.Unix(int64(file.UpdatedAt), 0),
			Size:      file.Size,
		})
	}

	return out, nil
}

func (s *GrpcStorage) File(ctx context.Context, user *models.User, fullpath string) (storage.File, error) {
	reply, err := s.Client.OpenFile(ctx,
		&OpenFileQuery{
			User: &User{
				Id: user.ID,
			},
			FullPath: fullpath,
		},
	)
	if err != nil {
		return nil, err
	}

	file := &File{
		Storage: s,
		Id:      reply.Id,
	}

	s.openFiles[reply.Id] = file

	return file, nil
}

func (s *GrpcStorage) Delete(ctx context.Context, user *models.User, fullpath string) error {
	err, Err := s.Client.Delete(ctx,
		&DeleteQuery{
			User: &User{
				Id: user.ID,
			},
			FullPath: fullpath,
		},
	)
	return toError2(err, Err)
}

func (s *GrpcStorage) closeFile(id uint64) {
	delete(s.openFiles, id)
}
