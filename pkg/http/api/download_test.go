package api

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage/builtin/memory"
	"github.com/stretchr/testify/assert"
)

func TestDownload(t *testing.T) {
	user := &models.User{
		ID: 1337,
	}

	memfs := memory.NewStorageProvider()

	handler := &downloadHandler{
		Storage: memfs,
	}

	data := &bytes.Buffer{}
	const length = 49569
	io.CopyN(data, rand.New(rand.NewSource(time.Now().UnixNano())), length)
	raw := data.Bytes()

	memfs.Data["/test.data"] = raw

	req, err := http.NewRequest(http.MethodGet, "/api/download?filename=test.data", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler.Serve(user, rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, rr.Result().Status)

	read, err := io.ReadAll(rr.Body)
	if assert.NoError(t, err) {
		assert.Equal(t, raw, read)
	}
}
