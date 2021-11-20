package fileinfo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/schoentoon/go-cloud/pkg/fileinfo/types"
	"github.com/schoentoon/go-cloud/pkg/storage"
)

type Manager struct {
	providers        map[string]types.FileInfoProvider
	mimeTypeProvider types.MimeTypeProvider
}

type Options struct {
	Render bool
}

func NewManager(mimetypeProvider string, provider ...string) (*Manager, error) {
	out := &Manager{
		providers: map[string]types.FileInfoProvider{},
	}

	mp, err := types.GetMimeProvider(mimetypeProvider)
	if err != nil {
		return nil, err
	}
	out.mimeTypeProvider = mp

	for _, name := range provider {
		p, err := types.GetProvider(name)
		if err != nil {
			return nil, err
		}
		out.providers[name] = p
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
		return nil, errors.New("No specified providers to check with")
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

		providers[name] = p

		if i == 0 {
			min = p.MinimumBytes(mime.Type, mime.SubType)
		} else if min != -1 {
			newMin := p.MinimumBytes(mime.Type, mime.SubType)
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

		tasks = append(tasks, &checkTask{
			reader:       rp,
			provider:     provider,
			providerName: name,
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
		io.Copy(mw, reader)

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
				out.Data[result.name] = types.Result{Data: provider.Render(result.data)}
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
}

type taskOut struct {
	data interface{}
	err  error
	name string
}

func (t *checkTask) Run(filename string, wg *sync.WaitGroup) (out interface{}, err error) {
	defer wg.Done()

	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = er
			}
		}
	}()

	defer io.Copy(io.Discard, t.reader)
	out, err = t.provider.Check(filename, t.reader)
	return
}
