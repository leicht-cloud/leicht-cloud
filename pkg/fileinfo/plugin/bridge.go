package plugin

// This file contains the implementation of the grpc to storageprovider interface bridge
// your plugin is really just a grpc server, that the go-cloud server will communicate with
// This struct here just make it very easy for you to set up the grpc server with a storageprovider interface

import (
	context "context"
	"io"

	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
)

type BridgeFileinfoProviderServer struct {
	FileInfo types.FileInfoProvider
}

func NewFileinfoBridge(fileinfo types.FileInfoProvider) *BridgeFileinfoProviderServer {
	return &BridgeFileinfoProviderServer{
		FileInfo: fileinfo,
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

func (s *BridgeFileinfoProviderServer) MinimumBytes(_ context.Context, req *MinimumBytesQuery) (*MinimumBytesResponse, error) {
	min, err := s.FileInfo.MinimumBytes(req.Type, req.Subtype)
	if err != nil {
		return &MinimumBytesResponse{
			Error: toError(err),
		}, nil
	}
	return &MinimumBytesResponse{
		Length: min,
	}, nil
}

func (s *BridgeFileinfoProviderServer) Check(srv FileInfoProvider_CheckServer) error {
	rp, wp := io.Pipe()
	errCh := make(chan error)
	filenameCh := make(chan string, 1)
	respCh := make(chan interface{})
	defer rp.Close()
	defer wp.Close()

	go func(filenameCh chan<- string, errCh chan<- error) {
		first := true
		for {
			msg, err := srv.Recv()
			if err != nil {
				if err == io.EOF {
					wp.Close()
				} else {
					errCh <- err
				}
				return
			}
			if msg.GetEOF() {
				wp.Close()
				return
			}

			if first {
				first = false
				filenameCh <- msg.GetFilename()
			}

			_, err = wp.Write(msg.GetData())
			if err != nil {
				errCh <- err
				return
			}
		}
	}(filenameCh, errCh)

	for {
		select {
		case err := <-errCh:
			return err
		case resp := <-respCh:
			if err, ok := resp.(error); ok {
				return srv.SendAndClose(&CheckResponse{
					Error: toError(err),
				})
			}

			if bytes, ok := resp.([]byte); ok {
				return srv.SendAndClose(&CheckResponse{
					Data: bytes,
				})
			}

			// we shouldn't ever reach this..
		case filename := <-filenameCh:
			go func(ch chan<- interface{}, filename string) {
				data, err := s.FileInfo.Check(filename, rp)
				if err != nil {
					ch <- err
				} else {
					ch <- data
				}
			}(respCh, filename)
		}
	}
}

func (s *BridgeFileinfoProviderServer) Render(_ context.Context, req *RenderQuery) (*RenderResponse, error) {
	content, title, err := s.FileInfo.Render(req.GetData())
	if err != nil {
		return &RenderResponse{
			Error: toError(err),
		}, nil
	}
	return &RenderResponse{
		Content: content,
		Title:   title,
	}, nil
}

func (s *BridgeFileinfoProviderServer) mustEmbedUnimplementedFileInfoProviderServer() {
}
