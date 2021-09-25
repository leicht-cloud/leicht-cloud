package plugin

import (
	"context"
	"errors"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
)

type BridgeStorageProviderServer struct {
	Storage storage.StorageProvider
}

func toError(err error) *Error {
	if err != nil {
		return &Error{
			Message: err.Error(),
		}
	}
	return nil
}

func toUser(req *User) *models.User {
	if req == nil {
		return nil
	}
	return &models.User{
		ID: req.GetId(),
	}
}

var ErrNoUser = errors.New("No user specified")

func (s *BridgeStorageProviderServer) InitUser(ctx context.Context, req *User) (*Error, error) {
	user := toUser(req)
	if user == nil {
		return nil, ErrNoUser
	}
	return toError(s.Storage.InitUser(ctx, user)), nil
}
func (s *BridgeStorageProviderServer) MkDir(ctx context.Context, req *MkdirQuery) (*Error, error) {
	user := toUser(req.GetUser())
	if user == nil {
		return nil, ErrNoUser
	}
	return toError(s.Storage.Mkdir(ctx, user, req.GetPath())), nil
}
func (s *BridgeStorageProviderServer) Move(ctx context.Context, req *MoveQuery) (*Error, error) {
	user := toUser(req.GetUser())
	if user == nil {
		return nil, ErrNoUser
	}
	return toError(s.Storage.Move(ctx, user, req.GetSrc(), req.GetDst())), nil
}
func (s *BridgeStorageProviderServer) ListDirectory(ctx context.Context, req *ListDirectoryQuery) (*ListDirectoryInfoReply, error) {
	user := toUser(req.GetUser())
	if user == nil {
		return nil, ErrNoUser
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
	return nil, errors.New("method OpenFile not implemented")
}

func (s *BridgeStorageProviderServer) CloseFile(ctx context.Context, req *CloseFileQuery) (*Error, error) {
	return nil, errors.New("method CloseFile not implemented")
}

func (s *BridgeStorageProviderServer) WriteFile(srv StorageProvider_WriteFileServer) error {
	return errors.New("method WriteFile not implemented")
}

func (s *BridgeStorageProviderServer) ReadFile(req *ReadFileQuery, srv StorageProvider_ReadFileServer) error {
	return errors.New("method ReadFile not implemented")
}
