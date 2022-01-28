package prometheus

import (
	"io"
	"time"

	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
	"github.com/prometheus/client_golang/prometheus"
)

type wrappedFileInfo struct {
	fileinfo types.FileInfoProvider

	promCheck, promRender *prometheus.SummaryVec
	name                  string
}

func (m *Manager) newWrappedFileInfo(fileinfo types.FileInfoProvider, name string) types.FileInfoProvider {
	return &wrappedFileInfo{
		fileinfo:   fileinfo,
		promCheck:  m.promCheck,
		promRender: m.promRender,
		name:       name,
	}
}

func (w *wrappedFileInfo) MinimumBytes(typ, subtyp string) (int64, error) {
	return w.fileinfo.MinimumBytes(typ, subtyp)
}

func (w *wrappedFileInfo) Check(filename string, reader io.Reader) ([]byte, error) {
	start := time.Now()
	out, err := w.fileinfo.Check(filename, reader)
	took := time.Since(start)

	w.promCheck.WithLabelValues(w.name).Observe(float64(took) / float64(time.Second))

	return out, err
}

func (w *wrappedFileInfo) Render(data []byte) (string, string, error) {
	start := time.Now()
	str1, str2, err := w.fileinfo.Render(data)
	took := time.Since(start)

	w.promRender.WithLabelValues(w.name).Observe(float64(took) / float64(time.Second))

	return str1, str2, err
}
