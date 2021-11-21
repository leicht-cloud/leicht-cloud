package fileinfo

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	fileinfoPlugin "github.com/schoentoon/go-cloud/pkg/fileinfo/plugin"
	"github.com/schoentoon/go-cloud/pkg/fileinfo/types"
	"github.com/schoentoon/go-cloud/pkg/plugin"
	"github.com/schoentoon/go-cloud/pkg/storage"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	providers        map[string]types.FileInfoProvider
	mimeTypeProvider types.MimeTypeProvider
}

type Options struct {
	Render bool
}

func NewManager(pManager *plugin.Manager, mimetypeProvider string, provider ...string) (*Manager, error) {
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
			conn, err := pManager.Start(name)
			if err != nil {
				return nil, err
			}

			provider, err := fileinfoPlugin.NewGrpcFileinfo(conn)
			if err != nil {
				return nil, err
			}
			out.providers[name] = provider
		} else {
			p, err := types.GetProvider(name)
			if err != nil {
				return nil, err
			}
			out.providers[name] = p
		}
	}

	return out, nil
}

type Output struct {
	Data     map[string]types.Result `json:"data"`
	MimeType types.MimeType          `json:"mime"`
	Filename string                  `json:"filename"`
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

	out := &Output{
		Data:     make(map[string]types.Result, len(providers)),
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

	wg := sync.WaitGroup{}
	wg.Add(len(tasks))
	outCh := make(chan taskOut, len(providers))

	for _, t := range tasks {
		go func(ch chan<- taskOut, t *checkTask) {
			data, err := t.Run(filename, &wg)
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

	for i := 0; i < len(providers); i++ {
		result := <-outCh
		if result.err != nil {
			out.Data[result.name] = types.Result{Err: result.err}
		} else {
			if opts.Render {
				provider, ok := m.providers[result.name]
				if !ok {
					return nil, fmt.Errorf("No provider found called: %s", result.name)
				}
				str, err := provider.Render(result.data)
				if err == nil {
					out.Data[result.name] = types.Result{Human: str, Data: result.data}
				} else {
					logrus.Error(err)
				}
			} else {
				out.Data[result.name] = types.Result{Data: result.data}
			}
		}
	}

	wg.Wait()

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

func (t *checkTask) Run(filename string, wg *sync.WaitGroup) (out []byte, err error) {
	defer wg.Done()

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
