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
)

type BridgeStorageProviderServer struct {
	Storage storage.StorageProvider

	mutex sync.RWMutex
	// the value of this map is always going to be storage.File,
	// but for some reason golang doesn't allow you to use this as a value?
	openFiles map[int]interface{}
	nextID    int
}

func NewStorageBridge(storage storage.StorageProvider) *BridgeStorageProviderServer {
	return &BridgeStorageProviderServer{
		Storage:   storage,
		openFiles: make(map[int]interface{}),
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

func (s *BridgeStorageProviderServer) ListDirectory(ctx context.Context, req *ListDirectoryQuery) (*ListDirectoryInfoReply, error) {
	user := toUser(req.GetUser())
	if user == nil {
		return &ListDirectoryInfoReply{
			Error: ErrNoUser,
		}, nil
	}

	files, err := s.Storage.ListDirectory(ctx, user, req.GetPath())
	if err != nil {
		return &ListDirectoryInfoReply{
			Error: toError(err),
		}, nil
	}

	out := &ListDirectoryInfoReply{
		Path:  req.GetPath(),
		Files: make([]*FileInfo, 0, len(files.Files)),
	}

	for _, f := range files.Files {
		file := &FileInfo{
			Name:      f.Name,
			FullPath:  f.FullPath,
			MimeType:  f.MimeType,
			CreatedAt: uint64(f.CreatedAt.Unix()),
			UpdatedAt: uint64(f.UpdatedAt.Unix()),
			Size:      f.Size,
		}

		out.Files = append(out.Files, file)
	}

	return out, nil
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
	fileId := s.nextID
	s.openFiles[fileId] = file
	s.nextID++

	return &OpenFileReply{
		Id: uint64(fileId),
	}, nil
}

func (s *BridgeStorageProviderServer) getFile(id int) (storage.File, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	file, ok := s.openFiles[id]
	if !ok {
		return nil, fmt.Errorf("No file with id: %d", id)
	}

	return file.(storage.File), nil
}

func (s *BridgeStorageProviderServer) CloseFile(ctx context.Context, req *CloseFileQuery) (*Error, error) {
	file, err := s.getFile(int(req.GetId()))
	if err != nil {
		return nil, err
	}

	// close the actual underlying file
	err = file.Close()
	if err != nil {
		return toError(err), nil
	}

	// and remove file from our open files map
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.openFiles, int(req.GetId()))

	return nil, nil
}

func (s *BridgeStorageProviderServer) WriteFile(srv StorageProvider_WriteFileServer) error {
	for {
		req, err := srv.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		file, err := s.getFile(int(req.GetId()))
		if err != nil {
			return err
		}

		_, err = file.Write(req.GetData())
		if err != nil {
			return err
		}
	}
}

const READ_BUFFER_SIZE = 1024 * 4

func (s *BridgeStorageProviderServer) ReadFile(req *ReadFileQuery, srv StorageProvider_ReadFileServer) error {
	file, err := s.getFile(int(req.GetId()))
	if err != nil {
		return err
	}

	buffer := make([]byte, READ_BUFFER_SIZE)

	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				// empty message indicate end of file
				// TODO: figure out if we can also just return from here instead
				return srv.Send(&ReadFileReply{})
			}
			return srv.Send(&ReadFileReply{
				Error: toError(err),
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
