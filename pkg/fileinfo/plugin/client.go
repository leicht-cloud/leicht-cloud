package plugin

// This is the client used internally to connect with plugins that make use of the BridgeStorageProviderServer
// which is located in bridge.go

import (
	"context"
	"errors"
	"io"

	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

type GrpcFileinfo struct {
	Conn   *grpc.ClientConn
	Client FileInfoProviderClient
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
		Conn:   conn,
		Client: NewFileInfoProviderClient(conn),
	}

	return out, nil
}

func (s *GrpcFileinfo) Close() error {
	return nil
}

func (s *GrpcFileinfo) MinimumBytes(typ, subtyp string) (int64, error) {
	resp, err := s.Client.MinimumBytes(context.Background(), &MinimumBytesQuery{
		Type:    typ,
		Subtype: subtyp,
	})
	err = toError2(resp.GetError(), err)
	if err != nil {
		return 0, err
	}

	return resp.GetLength(), nil
}

// TODO: We'll probably want to increase this, perhaps even have it more dynamic or configurable per plugin?
const READ_BUFFER_SIZE = 1024 * 4

func (s *GrpcFileinfo) Check(filename string, reader io.Reader) ([]byte, error) {
	client, err := s.Client.Check(context.Background())
	if err != nil {
		return nil, err
	}

	recvErrCh := make(chan error, 1)
	respErrCh := make(chan error, 1)
	respCh := make(chan *CheckResponse, 1)

	go func(errCh chan<- error) {
		buf := make([]byte, READ_BUFFER_SIZE)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					err = client.CloseSend()
					if err == nil {
						return
					}
				}
				logrus.Error(err)
				errCh <- err
				return
			}

			err = client.Send(&CheckQuery{
				Filename: filename,
				Data:     buf[:n],
			})
			if err != nil {
				if err == io.EOF {
					return
				}
				logrus.Error(err)
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
			logrus.Error(err)
			errCh <- err
			return
		}
		respCh <- resp
	}(respCh, respErrCh)

	for {
		select {
		case resp := <-respCh:
			logrus.Debugf("resp: %#v", resp)
			err := toError2(resp.GetError(), nil)
			if err != nil {
				return nil, err
			}
			return resp.GetData(), nil
		case err := <-recvErrCh:
			logrus.Debugf("err: %s", err)
			return nil, err
		case err := <-respErrCh:
			logrus.Debugf("err: %s", err)
			return nil, err
		}
	}
}

func (s *GrpcFileinfo) Render(data []byte) (string, error) {
	resp, err := s.Client.Render(context.Background(), &RenderQuery{Data: data})
	err = toError2(resp.GetError(), err)
	if err != nil {
		return "", err
	}

	return resp.GetOutput(), nil
}
