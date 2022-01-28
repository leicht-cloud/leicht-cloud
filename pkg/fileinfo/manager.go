package fileinfo

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	fileinfoPlugin "github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/fileinfo/types"
	"github.com/leicht-cloud/leicht-cloud/pkg/plugin"
	"github.com/leicht-cloud/leicht-cloud/pkg/prometheus"
	"github.com/leicht-cloud/leicht-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	providers        map[string]types.FileInfoProvider
	mimeTypeProvider types.MimeTypeProvider
}

type Options struct {
	Render bool
}

func NewManager(pManager *plugin.Manager, prom *prometheus.Manager, mimetypeProvider string, provider ...string) (*Manager, error) {
	out := &Manager{
		providers: map[string]types.FileInfoProvider{},
	}

	mp, err := types.GetMimeProvider(mimetypeProvider)
	if err != nil {
		return nil, err
	}
	out.mimeTypeProvider = mp

	for _, name := range provider {
		if strings.HasPrefix(name, "plugin:") {
			name = strings.TrimPrefix(name, "plugin:")
			plugin, err := pManager.Start(name)
			if err != nil {
				return nil, err
			}

			conn, err := plugin.GrpcConn()
			if err != nil {
				return nil, err
			}

			provider, err := fileinfoPlugin.NewGrpcFileinfo(conn)
			if err != nil {
				return nil, err
			}
			out.providers[name] = prom.WrapFileInfo(provider, name)
		} else {
			p, err := types.GetProvider(name)
			if err != nil {
				return nil, err
			}
			out.providers[name] = prom.WrapFileInfo(p, name)
		}
	}

	return out, nil
}

type Output struct {
	Channel  <-chan types.Result `json:"-"`
	MimeType types.MimeType      `json:"mime"`
	Filename string              `json:"filename"`
}

func (m *Manager) FileInfo(filename string, file storage.File, opts *Options, requestedProviders ...string) (*Output, error) {
	if len(requestedProviders) == 0 {
		// if there are no specific providers requested, we use all of them.
		for p := range m.providers {
			requestedProviders = append(requestedProviders, p)
		}
	}

	mimereader := io.LimitReader(file, m.mimeTypeProvider.MinimumBytes())
	mime, reader, err := m.readMime(filename, mimereader)
	if err != nil {
		return nil, err
	}
	reader = io.MultiReader(reader, file)

	providers := make(map[string]types.FileInfoProvider, len(requestedProviders))
	var min int64
	for i, name := range requestedProviders {
		p, ok := m.providers[name]
		if !ok {
			return nil, fmt.Errorf("No provider found called: %s", name)
		}

		pmin, err := p.MinimumBytes(mime.Type, mime.SubType)
		if err != nil {
			continue
		}

		providers[name] = p

		if i == 0 {
			min = pmin
		} else if min != -1 {
			newMin := pmin
			if newMin == -1 {
				min = -1
			} else if newMin > min {
				min = newMin
			}
		}
	}

	if min > 0 {
		reader = io.LimitReader(reader, min)
	}

	ch := make(chan types.Result, len(providers)+1) // the + 1 is for the mime type
	out := &Output{
		Channel:  ch,
		MimeType: *mime,
		Filename: filename,
	}
	tasks := make([]*checkTask, 0, len(providers))
	writers := make([]io.Writer, 0, len(providers))
	closers := make([]io.Closer, 0, len(providers))

	for name, provider := range providers {
		rp, wp := io.Pipe()
		writers = append(writers, wp)
		closers = append(closers, wp)

		pmin, err := provider.MinimumBytes(mime.Type, mime.SubType)
		if err != nil {
			continue
		}

		tasks = append(tasks, &checkTask{
			reader:       rp,
			provider:     provider,
			providerName: name,
			minbytes:     pmin,
		})
	}

	outCh := make(chan taskOut, len(providers))

	for _, t := range tasks {
		go func(ch chan<- taskOut, t *checkTask) {
			data, err := t.Run(filename)
			ch <- taskOut{data: data, err: err, name: t.providerName}
		}(outCh, t)
	}

	go func() {
		mw := io.MultiWriter(writers...)
		_, err := io.Copy(mw, reader)
		if err != nil {
			logrus.Error(err)
		}

		for _, c := range closers {
			c.Close()
		}
	}()

	ch <- types.Result{
		Name:  "mime",
		Data:  []byte(mime.String()),
		Human: mime.String(),
		Err:   nil,
	}

	go func(ch chan<- types.Result) {
		for i := 0; i < len(providers); i++ {
			result := <-outCh
			if result.err != nil {
				ch <- types.Result{
					Name: result.name,
					Err:  result.err,
				}
			} else {
				if opts.Render {
					provider, ok := providers[result.name]
					if !ok {
						logrus.Errorf("No provider found called: %s", result.name)
						continue
					}
					content, title, err := provider.Render(result.data)
					if err == nil {
						ch <- types.Result{
							Name:  result.name,
							Human: content,
							Title: title,
							Data:  result.data,
						}
					} else {
						logrus.Error(err)
					}
				} else {
					ch <- types.Result{
						Name: result.name,
						Data: result.data,
					}
				}
			}
		}

		close(ch)
	}(ch)

	return out, nil
}

func (m *Manager) readMime(filename string, reader io.Reader) (*types.MimeType, io.Reader, error) {
	buf, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}

	mime, err := m.mimeTypeProvider.MimeType(filename, bytes.NewReader(buf))
	if err != nil {
		return nil, nil, err
	}

	return mime, bytes.NewReader(buf), nil
}

type checkTask struct {
	reader       io.Reader
	provider     types.FileInfoProvider
	providerName string
	minbytes     int64
}

type taskOut struct {
	data []byte
	err  error
	name string
}

func (t *checkTask) Run(filename string) (out []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = er
			}
		}
	}()

	reader := t.reader
	if t.minbytes > 0 {
		reader = io.LimitReader(reader, t.minbytes)
	}

	defer func() {
		_, err := io.Copy(io.Discard, t.reader) // it is important here that we drain the full io.Reader, not the possibly limited one
		if err != nil {
			logrus.Error(err)
		}
	}()

	out, err = t.provider.Check(filename, reader)
	return
}
