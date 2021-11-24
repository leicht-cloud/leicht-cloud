package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/schoentoon/go-cloud/pkg/models"
	"github.com/schoentoon/go-cloud/pkg/storage"
)

type wrappedStorage struct {
	store storage.StorageProvider

	promInitUser, promMkdir, promMove, promListDirectory, promDelete *prometheus.SummaryVec
	promFilesOpen, promFileRead, promFileWrite                       *prometheus.GaugeVec
}

func newWrappedStorage(store storage.StorageProvider) *wrappedStorage {
	out := &wrappedStorage{
		store: store,
		promInitUser: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "init_user",
			}, nil,
		),
		promMkdir: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "mkdir",
			}, nil,
		),
		promMove: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "move",
			}, nil,
		),
		promListDirectory: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "list_directory",
			}, nil,
		),
		promDelete: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "delete",
			}, nil,
		),
		promFilesOpen: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "open_files",
			}, nil,
		),
		promFileRead: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "file_read_bytes",
			}, nil,
		),
		promFileWrite: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "file_write_bytes",
			}, nil,
		),
	}

	registry := prometheus.WrapRegistererWithPrefix("storage_", prometheus.DefaultRegisterer)

	registry.MustRegister(out.promInitUser,
		out.promMkdir,
		out.promMove,
		out.promListDirectory,
		out.promDelete,
		out.promFilesOpen,
		out.promFileRead,
		out.promFileWrite,
	)

	return out
}

func (w *wrappedStorage) InitUser(ctx context.Context, user *models.User) error {
	start := time.Now()
	err := w.store.InitUser(ctx, user)
	took := time.Since(start)

	w.promInitUser.With(nil).Observe(float64(took) / float64(time.Second))

	return err
}

func (w *wrappedStorage) Mkdir(ctx context.Context, user *models.User, path string) error {
	start := time.Now()
	err := w.store.Mkdir(ctx, user, path)
	took := time.Since(start)

	w.promMkdir.With(nil).Observe(float64(took) / float64(time.Second))

	return err
}

func (w *wrappedStorage) Move(ctx context.Context, user *models.User, src, dst string) error {
	start := time.Now()
	err := w.store.Move(ctx, user, src, dst)
	took := time.Since(start)

	w.promMove.With(nil).Observe(float64(took) / float64(time.Second))

	return err
}

func (w *wrappedStorage) ListDirectory(ctx context.Context, user *models.User, path string) (<-chan storage.FileInfo, error) {
	start := time.Now()
	ch, err := w.store.ListDirectory(ctx, user, path)
	took := time.Since(start)

	w.promListDirectory.With(nil).Observe(float64(took) / float64(time.Second))

	return ch, err
}

func (w *wrappedStorage) File(ctx context.Context, user *models.User, fullpath string) (storage.File, error) {
	file, err := w.store.File(ctx, user, fullpath)
	if file != nil {
		w.promFilesOpen.With(nil).Inc()
		return &wrappedFile{
			file:  file,
			open:  w.promFilesOpen,
			read:  w.promFileRead,
			write: w.promFileWrite,
		}, nil
	}

	return nil, err
}

func (w *wrappedStorage) Delete(ctx context.Context, user *models.User, fullpath string) error {
	start := time.Now()
	err := w.store.Delete(ctx, user, fullpath)
	took := time.Since(start)

	w.promDelete.With(nil).Observe(float64(took) / float64(time.Second))

	return err
}

type wrappedFile struct {
	file storage.File

	open, read, write *prometheus.GaugeVec
}

func (w *wrappedFile) Close() error {
	w.open.With(nil).Dec()
	return w.file.Close()
}

func (w *wrappedFile) Read(d []byte) (int, error) {
	n, err := w.file.Read(d)
	if err == nil {
		w.read.With(nil).Add(float64(n))
	}

	return n, err
}

func (w *wrappedFile) Write(d []byte) (int, error) {
	n, err := w.file.Write(d)
	if err == nil {
		w.write.With(nil).Add(float64(n))
	}

	return n, err
}
