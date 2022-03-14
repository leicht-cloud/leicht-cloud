package webapi

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leicht-cloud/leicht-cloud/pkg/models"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage/builtin/memory"
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
	_, err := io.CopyN(data, rand.New(rand.NewSource(time.Now().UnixNano())), length)
	assert.NoError(t, err, err)
	raw := data.Bytes()

	memfs.Data["/some/nested/path/test.data"] = raw

	req, err := http.NewRequest(http.MethodGet, "/webapi/download?filename=/some/nested/path/test.data", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler.Serve(user, rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, rr.Result().Status)
	assert.Equal(t, "attachment; filename=\"test.data\"", rr.Header().Get("Content-Disposition"))

	read, err := io.ReadAll(rr.Body)
	if assert.NoError(t, err) {
		assert.Equal(t, raw, read)
	}
}
