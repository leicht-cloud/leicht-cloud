package plugin

// This is the client used internally to connect with plugins that make use of the BridgeStorageProviderServer
// which is located in bridge.go

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
	"github.com/leicht-cloud/leicht-cloud/pkg/system"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

type GrpcFileinfo struct {
	Conn   *grpc.ClientConn
	Client FileInfoProviderClient

	mutex    sync.RWMutex
	minBytes map[string]int64
	skipMap  map[string]struct{}
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

func NewGrpcFileinfo(conn *grpc.ClientConn) (*GrpcFileinfo, error) {
	out := &GrpcFileinfo{
		Conn:     conn,
		Client:   NewFileInfoProviderClient(conn),
		minBytes: make(map[string]int64),
		skipMap:  make(map[string]struct{}),
	}

	return out, nil
}

func (s *GrpcFileinfo) Close() error {
	return nil
}

func (s *GrpcFileinfo) cachedMinBytes(key string) (int64, error, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, ok := s.skipMap[key]
	if ok {
		return 0, types.ErrSkip, true
	}

	min, ok := s.minBytes[key]
	return min, nil, ok
}

func (s *GrpcFileinfo) MinimumBytes(typ, subtyp string) (int64, error) {
	key := fmt.Sprintf("%s/%s", typ, subtyp)
	min, err, ok := s.cachedMinBytes(key)
	if ok {
		if err != nil {
			return 0, err
		}
		return min, nil
	}

	resp, err := s.Client.MinimumBytes(context.Background(), &MinimumBytesQuery{
		Type:    typ,
		Subtype: subtyp,
	})
	err = toError2(resp.GetError(), err)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err != nil {
		s.skipMap[key] = struct{}{}
		return 0, err
	}

	s.minBytes[key] = resp.GetLength()
	return resp.GetLength(), nil
}

func (s *GrpcFileinfo) Check(filename string, reader io.Reader) ([]byte, error) {
	client, err := s.Client.Check(context.Background())
	if err != nil {
		return nil, err
	}

	recvErrCh := make(chan error, 1)
	respErrCh := make(chan error, 1)
	respCh := make(chan *CheckResponse, 1)

	go func(errCh chan<- error) {
		buf := make([]byte, system.GetBufferSize())
		logrus.Debugf("buffer size: %d", len(buf))
		for {
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					_ = client.Send(&CheckQuery{
						Filename: filename,
						Data:     buf[:n],
						EOF:      true,
					})
					return
				} else {
					errCh <- err
					return
				}
			}

			err = client.Send(&CheckQuery{
				Filename: filename,
				Data:     buf[:n],
			})
			if err != nil {
				errCh <- err
				return
			}
		}
	}(recvErrCh)

	go func(respCh chan<- *CheckResponse, errCh chan<- error) {
		// we can't use CloseAndRecv() here.. as this will close our client right away, lol
		resp := &CheckResponse{}
		err := client.RecvMsg(resp)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}(respCh, respErrCh)

	for {
		select {
		case resp := <-respCh:
			err := toError2(resp.GetError(), nil)
			if err != nil {
				return nil, err
			}
			return resp.GetData(), nil
		case err := <-recvErrCh:
			return nil, err
		case err := <-respErrCh:
			return nil, err
		}
	}
}

func (s *GrpcFileinfo) Render(data []byte) (string, string, error) {
	resp, err := s.Client.Render(context.Background(), &RenderQuery{Data: data})
	err = toError2(resp.GetError(), err)
	if err != nil {
		return "", "", err
	}

	return resp.GetContent(), resp.GetTitle(), nil
}
