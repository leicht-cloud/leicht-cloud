package plugin

// This file contains the implementation of the grpc to storageprovider interface bridge
// your plugin is really just a grpc server, that the go-cloud server will communicate with
// This struct here just make it very easy for you to set up the grpc server with a storageprovider interface

import (
	context "context"
	"io"
	"sync"

	"github.com/schoentoon/go-cloud/pkg/fileinfo/types"
	"github.com/sirupsen/logrus"
)

type BridgeFileinfoProviderServer struct {
	FileInfo types.FileInfoProvider

	mutex sync.RWMutex
	// the value of this map is always going to be storage.File,
	// but for some reason golang doesn't allow you to use this as a value?
	openFiles map[int32]interface{}
	nextID    int32
}

func NewFileinfoBridge(fileinfo types.FileInfoProvider) *BridgeFileinfoProviderServer {
	return &BridgeFileinfoProviderServer{
		FileInfo:  fileinfo,
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

func (s *BridgeFileinfoProviderServer) MinimumBytes(_ context.Context, req *MinimumBytesQuery) (*MinimumBytesResponse, error) {
	min, err := s.FileInfo.MinimumBytes(req.Type, req.Subtype)
	logrus.Debugf("min: %d, err: %s", min, err)
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
	recvCh := make(chan *CheckQuery)
	respCh := make(chan interface{})
	defer rp.Close()
	defer wp.Close()

	go func(recvCh chan<- *CheckQuery, errCh chan<- error) {
		for {
			msg, err := srv.Recv()
			if err != nil {
				errCh <- err
				return
			}
			recvCh <- msg
		}
	}(recvCh, errCh)

	first := true
	for {
		select {
		case err := <-errCh:
			logrus.Debugf("err: %s", err)
			if err == io.EOF {
				wp.Close()
				break
			}
			return err
		case resp := <-respCh:
			logrus.Debugf("resp: %#v", resp)
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
		case msg := <-recvCh:
			logrus.Debugf("recv: len(%d)", len(msg.GetData()))
			if first {
				first = false
				go func(ch chan<- interface{}, filename string) {
					data, err := s.FileInfo.Check(filename, rp)
					logrus.Debugf("data: %s, err: %s", data, err)
					if err != nil {
						ch <- err
					} else {
						ch <- data
					}
				}(respCh, msg.GetFilename())
			}
			_, err := wp.Write(msg.GetData())
			if err != nil {
				return err
			}
		}
	}
}

func (s *BridgeFileinfoProviderServer) Render(_ context.Context, req *RenderQuery) (*RenderResponse, error) {
	logrus.Debugf("Render(%#v)", req.GetData())
	out, err := s.FileInfo.Render(req.GetData())
	logrus.Debugf("out: %s, err: %s", out, err)
	if err != nil {
		return &RenderResponse{
			Error: toError(err),
		}, nil
	}
	return &RenderResponse{
		Output: out,
	}, nil
}

func (s *BridgeFileinfoProviderServer) mustEmbedUnimplementedFileInfoProviderServer() {
}
