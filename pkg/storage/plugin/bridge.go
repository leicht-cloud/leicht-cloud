package plugin

// This file contains the implementation of the grpc to storageprovider interface bridge
// your plugin is really just a grpc server, that the go-cloud server will communicate with
// This struct here just make it very easy for you to set up the grpc server with a storageprovider interface

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type BridgeStorageProviderServer struct {
	Storage storage.StorageProvider

	mutex sync.RWMutex
	// the value of this map is always going to be storage.File,
	// but for some reason golang doesn't allow you to use this as a value?
	openFiles map[int32]interface{}
	nextID    int32
}

func NewStorageBridge(storage storage.StorageProvider) *BridgeStorageProviderServer {
	return &BridgeStorageProviderServer{
		Storage:   storage,
		openFiles: make(map[int32]interface{}),
	}
}

func toError(err error) *Error {
	if err != nil {
		return &Error{
			Message: err.Error(),
		}
	}
	return &Error{}
}

func toUser(req *User) *models.User {
	if req == nil {
		return nil
	}
	return &models.User{
		ID: req.GetId(),
	}
}

var ErrNoUser = &Error{Message: "No user specified"}

func (s *BridgeStorageProviderServer) Configure(ctx context.Context, req *ConfigData) (*Error, error) {
	err := yaml.Unmarshal(req.GetYaml(), s.Storage)
	if err != nil {
		return toError(err), nil
	}

	// if the underlying storage has the OnConfigure function we call that before returning
	if onconfig, ok := s.Storage.(storage.PostConfigure); ok {
		return toError(onconfig.OnConfigure()), nil
	}

	return &Error{}, nil
}

func (s *BridgeStorageProviderServer) InitUser(ctx context.Context, req *User) (*Error, error) {
	user := toUser(req)
	if user == nil {
		return ErrNoUser, nil
	}
	return toError(s.Storage.InitUser(ctx, user)), nil
}

func (s *BridgeStorageProviderServer) MkDir(ctx context.Context, req *MkdirQuery) (*Error, error) {
	user := toUser(req.GetUser())
	if user == nil {
		return ErrNoUser, nil
	}
	return toError(s.Storage.Mkdir(ctx, user, req.GetPath())), nil
}

func (s *BridgeStorageProviderServer) Move(ctx context.Context, req *MoveQuery) (*Error, error) {
	user := toUser(req.GetUser())
	if user == nil {
		return ErrNoUser, nil
	}
	return toError(s.Storage.Move(ctx, user, req.GetSrc(), req.GetDst())), nil
}

func (s *BridgeStorageProviderServer) ListDirectory(req *ListDirectoryQuery, srv StorageProvider_ListDirectoryServer) error {
	user := toUser(req.GetUser())
	if user == nil {
		return nil
	}

	files, err := s.Storage.ListDirectory(srv.Context(), user, req.GetPath())
	if err != nil {
		return err
	}

	for f := range files {
		err = srv.Send(&FileInfo{
			Name:      f.Name,
			FullPath:  f.FullPath,
			MimeType:  f.MimeType,
			CreatedAt: uint64(f.CreatedAt.Unix()),
			UpdatedAt: uint64(f.UpdatedAt.Unix()),
			Size:      f.Size,
			Directory: f.Directory,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *BridgeStorageProviderServer) OpenFile(ctx context.Context, req *OpenFileQuery) (*OpenFileReply, error) {
	user := toUser(req.GetUser())
	if user == nil {
		return &OpenFileReply{
			Error: ErrNoUser,
		}, nil
	}

	file, err := s.Storage.File(ctx, user, req.GetFullPath())
	if err != nil {
		return &OpenFileReply{
			Error: toError(err),
		}, nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.nextID++
	s.openFiles[s.nextID] = file

	return &OpenFileReply{
		Id: s.nextID,
	}, nil
}

func (s *BridgeStorageProviderServer) getFile(id int32) (storage.File, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	file, ok := s.openFiles[id]
	if !ok {
		return nil, fmt.Errorf("No file with id: %d", id)
	}

	return file.(storage.File), nil
}

func (s *BridgeStorageProviderServer) CloseFile(ctx context.Context, req *CloseFileQuery) (*Error, error) {
	file, err := s.getFile(req.GetId())
	if err != nil {
		return toError(err), nil
	}

	// close the actual underlying file
	err = file.Close()
	if err != nil {
		return toError(err), nil
	}

	// and remove file from our open files map
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.openFiles, req.GetId())

	return &Error{}, nil
}

func (s *BridgeStorageProviderServer) WriteFile(ctx context.Context, req *WriteFileQuery) (*WriteFileReply, error) {
	file, err := s.getFile(req.GetId())
	if err != nil {
		return &WriteFileReply{
			Error: toError(err),
		}, nil
	}

	n, err := file.Write(req.GetData())
	if err != nil {
		return &WriteFileReply{
			Error: toError(err),
		}, nil
	}

	return &WriteFileReply{
		SizeWritten: int32(n),
	}, nil
}

// TODO: We'll probably want to increase this, perhaps even have it more dynamic or configurable per plugin?
const READ_BUFFER_SIZE = 1024 * 4

func (s *BridgeStorageProviderServer) ReadFile(req *ReadFileQuery, srv StorageProvider_ReadFileServer) error {
	file, err := s.getFile(req.GetId())
	if err != nil {
		return err
	}

	buffer := make([]byte, READ_BUFFER_SIZE)

	for {
		n, err := file.Read(buffer)
		logrus.Debugf("err: %s, n: %d", err, n)
		if err != nil {
			if err == io.EOF {
				// empty message indicate end of file
				return srv.Send(&ReadFileReply{
					Data: buffer[:n],
					EOF:  true,
				})
			}
			return srv.Send(&ReadFileReply{
				Error: toError(err),
			})
		} else if n < 0 {
			return srv.Send(&ReadFileReply{
				Error: &Error{
					Message: fmt.Sprintf("Read returned %d bytes and no error???", n),
				},
			})
		}

		err = srv.Send(&ReadFileReply{
			Data: buffer[:n],
		})
		if err != nil {
			return err
		}
	}
}

func (s *BridgeStorageProviderServer) Delete(ctx context.Context, req *DeleteQuery) (*Error, error) {
	user := toUser(req.GetUser())
	if user == nil {
		return ErrNoUser, nil
	}
	return toError(s.Storage.Delete(ctx, user, req.GetFullPath())), nil
}

func (s *BridgeStorageProviderServer) mustEmbedUnimplementedStorageProviderServer() {
}
