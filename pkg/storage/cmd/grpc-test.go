package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"google.golang.org/grpc"

	"github.com/schoentoon/go-cloud/pkg/models"
	storagePlugin "github.com/schoentoon/go-cloud/pkg/storage/plugin"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	client, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", 65000),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		panic(err)
	}

	store, err := storagePlugin.NewGrpcStorage(client, map[interface{}]interface{}{})
	if err != nil {
		panic(err)
	}

	user := &models.User{ID: 1337}

	err = store.InitUser(context.Background(), user)
	if err != nil {
		panic(err)
	}

	dir, err := store.ListDirectory(context.Background(), user, "random/dir")
	if err != nil {
		panic(err)
	}
	logrus.Debugf("%+v", dir)

	f, err := store.File(context.Background(), user, "test-file")
	if err != nil {
		panic(err)
	}

	src := rand.New(rand.NewSource(time.Now().Unix()))

	buffered := bufio.NewWriter(f)
	_, err = io.CopyN(buffered, src, 1024)
	if err != nil {
		panic(err)
	}
	buffered.Flush()
	f.Close()

	f, err = store.File(context.Background(), user, "test-file")
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)

	n, err := io.Copy(buf, f)
	if err != nil {
		panic(err)
	}
	logrus.Debugf("Read %d bytes, buf is %d bytes long.", n, buf.Len())
}
