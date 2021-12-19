package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/schoentoon/go-cloud/pkg/storage/builtin/memory"
	"github.com/stretchr/testify/assert"
)

// This test is basically a direct copy of https://github.com/tus/tusd/blob/master/pkg/handler/unrouted_handler_test.go
func TestParseTusMetadata(t *testing.T) {
	md := parseTusMetadata("")
	assert.Equal(t, md, map[string]string{})

	// Invalidly encoded values are ignored
	md = parseTusMetadata("k1 INVALID")
	assert.Equal(t, md, map[string]string{})

	// If the same key occurs multiple times, the last one wins
	md = parseTusMetadata("k1 aGVsbG8=,k1 d29ybGQ=")
	assert.Equal(t, md, map[string]string{
		"k1": "world",
	})

	// Empty values are mapped to an empty string
	md = parseTusMetadata("k1 aGVsbG8=, k2, k3 , k4 d29ybGQ=")
	assert.Equal(t, md, map[string]string{
		"k1": "hello",
		"k2": "",
		"k3": "",
		"k4": "world",
	})
}

func initUploadHandler(t *testing.T) (storage.StorageProvider, *uploadHandler) {
	store := memory.NewStorageProvider()

	handler := &uploadHandler{
		Storage: store,
		uploads: make(map[int64]*uploadState),
	}

	return store, handler
}

func TestTusUploadFullCycle(t *testing.T) {
	store, handler := initUploadHandler(t)

	user := &models.User{
		ID: 1337,
	}

	data := &bytes.Buffer{}
	const length = 49569
	io.CopyN(data, rand.New(rand.NewSource(time.Now().UnixNano())), length)
	raw := data.Bytes()

	createReq, err := http.NewRequest(http.MethodPost, "/api/upload", nil)
	if err != nil {
		t.Fatal(err)
	}

	createReq.Header.Add("Upload-Length", fmt.Sprintf("%d", data.Len()))
	createReq.Header.Add("Upload-Metadata", "filename dGVzdC5kYXRh") // base64 for "test.data"

	createRR := httptest.NewRecorder()

	handler.Serve(user, createRR, createReq)

	assert.Equal(t, http.StatusCreated, createRR.Code, createRR.Result().Status)
	resume := createRR.Header().Get("Location")
	assert.NotEmpty(t, resume)

	pos := 0
	const chunkSize = 1024 * 4
	for {
		head, err := http.NewRequest(http.MethodHead, resume, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		handler.Serve(user, rr, head)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, fmt.Sprintf("%d", pos), rr.Header().Get("Upload-Offset"))
		assert.Equal(t, fmt.Sprintf("%d", length), rr.Header().Get("Upload-Length"))

		req, err := http.NewRequest(http.MethodPatch, resume, io.LimitReader(data, chunkSize))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Upload-Offset", fmt.Sprintf("%d", pos))

		rr = httptest.NewRecorder()

		handler.Serve(user, rr, req)

		if !assert.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String()) {
			break
		}

		pos += chunkSize
		if pos > length {
			pos = length
		}

		if !assert.Equal(t, fmt.Sprintf("%d", pos), rr.Header().Get("Upload-Offset")) {
			break
		}

		if pos >= length {
			break
		}
	}

	file, err := store.File(context.Background(), user, "/test.data")
	assert.NoError(t, err)

	written, err := ioutil.ReadAll(file)
	if assert.NoError(t, err) {
		assert.Equal(t, raw, written)
	}
}

func TestTusOptions(t *testing.T) {
	_, handler := initUploadHandler(t)

	user := &models.User{
		ID: 1337,
	}

	req, err := http.NewRequest(http.MethodOptions, "/api/upload", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler.Serve(user, rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, rr.Result().Status)
}

func TestTusInvalidLength(t *testing.T) {
	_, handler := initUploadHandler(t)

	user := &models.User{
		ID: 1337,
	}

	req, err := http.NewRequest(http.MethodPost, "/api/upload", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Upload-Length", "This is not a number, lol")
	req.Header.Add("Upload-Metadata", "filename dGVzdC5kYXRh") // base64 for "test.data"

	rr := httptest.NewRecorder()

	handler.Serve(user, rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, rr.Result().Status)
	assert.Empty(t, rr.Header().Get("Location"))
}

func TestTusInvalidFilename(t *testing.T) {
	_, handler := initUploadHandler(t)

	user := &models.User{
		ID: 1337,
	}

	req, err := http.NewRequest(http.MethodPost, "/api/upload", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Upload-Length", "9001")
	req.Header.Add("Upload-Metadata", "") // the metadata is just empty instead

	rr := httptest.NewRecorder()

	handler.Serve(user, rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, rr.Result().Status)
	assert.Empty(t, rr.Header().Get("Location"))
}

func TestTusResumeInvalidID(t *testing.T) {
	_, handler := initUploadHandler(t)

	user := &models.User{
		ID: 1337,
	}

	req, err := http.NewRequest(http.MethodPatch, "/api/upload?resume=ThisIsInvalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler.Serve(user, rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, rr.Result().Status)
}

func TestTusResumeNotExistingID(t *testing.T) {
	_, handler := initUploadHandler(t)

	user := &models.User{
		ID: 1337,
	}

	req, err := http.NewRequest(http.MethodPatch, "/api/upload?resume=9001", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler.Serve(user, rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code, rr.Result().Status)
}
