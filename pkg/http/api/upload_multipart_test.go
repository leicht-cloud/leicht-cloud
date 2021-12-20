package api

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestMultipart(t *testing.T) {
	store, handler := initUploadHandler(t)

	user := &models.User{
		ID: 1337,
	}

	data := &bytes.Buffer{}
	const length = 49569
	_, err := io.CopyN(data, rand.New(rand.NewSource(time.Now().UnixNano())), length)
	assert.NoError(t, err, err)
	raw := data.Bytes()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("", "test.data")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(part, data)
	assert.NoError(t, err, err)

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, "/api/upload", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler.Serve(user, rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code, rr.Result().Status)

	file, err := store.File(context.Background(), user, "/test.data")
	assert.NoError(t, err)

	written, err := ioutil.ReadAll(file)
	if assert.NoError(t, err) {
		assert.Equal(t, raw, written)
	}
}

func TestMultipartInvalidFilename(t *testing.T) {
	_, handler := initUploadHandler(t)

	user := &models.User{
		ID: 1337,
	}

	data := &bytes.Buffer{}
	const length = 49569
	_, err := io.CopyN(data, rand.New(rand.NewSource(0)), length)
	assert.NoError(t, err, err)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(part, data)
	assert.NoError(t, err, err)

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, "/api/upload", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	handler.Serve(user, rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, rr.Result().Status)
}
