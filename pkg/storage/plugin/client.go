package plugin

// This is the client used internally to connect with plugins that make use of the BridgeStorageProviderServer
// which is located in bridge.go

import (
	context "context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type GrpcStorage struct {
	Conn   *grpc.ClientConn
	Client StorageProviderClient

	mutex     sync.RWMutex
	openFiles map[int32]*File
}

func toError2(err *Error, Err error) error {
	if Err != nil {
		return Err
	}
	if err != nil && err.GetMessage() != "" {
		return errors.New(err.GetMessage())
	}
	return nil
}

func NewGrpcStorage(conn *grpc.ClientConn, config map[interface{}]interface{}) (*GrpcStorage, error) {
	out := &GrpcStorage{
		Conn:      conn,
		Client:    NewStorageProviderClient(conn),
		openFiles: make(map[int32]*File),
	}

	err := out.configure(config)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *GrpcStorage) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, f := range s.openFiles {
		f.Close()
	}
	return nil
}

func (s *GrpcStorage) configure(in map[interface{}]interface{}) error {
	data, err := yaml.Marshal(in)
	if err != nil {
		return err
	}

	Err, err := s.Client.Configure(context.Background(), &ConfigData{
		Yaml: data,
	})

	return toError2(Err, err)
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

func (s *GrpcStorage) ListDirectory(ctx context.Context, user *models.User, path string) (<-chan storage.FileInfo, error) {
	dir, err := s.Client.ListDirectory(ctx,
		&ListDirectoryQuery{
			User: &User{
				Id: user.ID,
			},
			Path: path,
		},
	)
	if err != nil {
		return nil, err
	}

	out := make(chan storage.FileInfo, 1)

	reply, err := dir.Recv()
	if err != nil {
		return nil, err
	} else {
		out <- storage.FileInfo{
			Name:      reply.Name,
			FullPath:  reply.FullPath,
			MimeType:  reply.MimeType,
			CreatedAt: time.Unix(int64(reply.CreatedAt), 0),
			UpdatedAt: time.Unix(int64(reply.UpdatedAt), 0),
			Size:      reply.Size,
		}
	}

	go func(out chan<- storage.FileInfo) {
		for {
			reply, err := dir.Recv()
			if err == io.EOF {
				close(out)
				return
			}

			out <- storage.FileInfo{
				Name:      reply.Name,
				FullPath:  reply.FullPath,
				MimeType:  reply.MimeType,
				CreatedAt: time.Unix(int64(reply.CreatedAt), 0),
				UpdatedAt: time.Unix(int64(reply.UpdatedAt), 0),
				Size:      reply.Size,
			}
		}
	}(out)

	return out, nil
}

func (s *GrpcStorage) File(ctx context.Context, user *models.User, fullpath string) (storage.File, error) {
	logrus.Debugf("File(%d, %s)", user.ID, fullpath)
	reply, err := s.Client.OpenFile(ctx,
		&OpenFileQuery{
			User: &User{
				Id: user.ID,
			},
			FullPath: fullpath,
		},
	)
	if err != nil {
		logrus.Errorf("%s opening %s for %d", err, fullpath, user.ID)
		return nil, err
	}

	file := &File{
		Storage: s,
		Id:      reply.Id,
	}

	logrus.Debugf("Opened %s for %d with id %d", fullpath, user.ID, reply.Id)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.openFiles[reply.Id] = file

	logrus.Debugf("%+v", s.openFiles)

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

func (s *GrpcStorage) closeFile(id int32) error {
	err, Err := s.Client.CloseFile(context.TODO(),
		&CloseFileQuery{
			Id: id,
		},
	)

	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.openFiles, id)

	return toError2(err, Err)
}
