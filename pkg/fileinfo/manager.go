package fileinfo

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/schoentoon/go-cloud/pkg/storage"
)

type Manager struct {
	providers map[string]FileInfoProvider
}

type Options struct {
	Render bool
}

func NewManager(provider ...string) (*Manager, error) {
	out := &Manager{
		providers: map[string]FileInfoProvider{},
	}

	for _, name := range provider {
		p, err := GetProvider(name)
		if err != nil {
			return nil, err
		}
		out.providers[name] = p
	}

	return out, nil
}

func (m *Manager) FileInfo(filename string, file storage.File, opts *Options, requestedProviders ...string) (map[string]Result, error) {
	if len(requestedProviders) == 0 {
		return nil, errors.New("No specified providers to check with")
	}

	providers := make(map[string]FileInfoProvider, len(requestedProviders))
	var min int64
	for i, name := range requestedProviders {
		p, ok := m.providers[name]
		if !ok {
			return nil, fmt.Errorf("No provider found called: %s", name)
		}

		providers[name] = p

		if i == 0 {
			min = p.MinimumBytes()
		} else if min != -1 {
			newMin := p.MinimumBytes()
			if newMin == -1 {
				min = -1
			} else if newMin > min {
				min = newMin
			}
		}
	}

	var reader io.Reader = file
	if min > 0 {
		reader = io.LimitReader(reader, min)
	}

	out := make(map[string]Result)
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
			out[result.name] = Result{Err: result.err}
		} else {
			if opts.Render {
				provider, ok := m.providers[result.name]
				if !ok {
					return nil, fmt.Errorf("No provider found called: %s", result.name)
				}
				out[result.name] = Result{Data: provider.Render(result.data)}
			} else {
				out[result.name] = Result{Data: result.data}
			}
		}
	}

	wg.Wait()

	return out, nil
}

type checkTask struct {
	reader       io.Reader
	provider     FileInfoProvider
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
