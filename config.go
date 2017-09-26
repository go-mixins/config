// Package config provides generic reloadable configuration for application
package config

import (
	"bytes"
	"crypto/sha256"
	"hash"
	"io"
	"os"
	"sync"
	"time"

	"github.com/go-fsnotify/fsnotify"
	"github.com/pkg/errors"
)

// Config represents hash-summed configuration file. It's being reloaded periodically to ensure
// that most recent version of the configuration is used.
type Config struct {
	name        string
	stop        chan struct{}
	forceReload chan bool
	hash        hash.Hash
	watcher     *fsnotify.Watcher
}

// New creates config reloader for a given file name. The decoder function
// must parse data from supplied reader or return error, which will be passed
// to onChange handler function. Underlying file is re-read on every fsnotify
// event, with 30-second forced reload as a fallback.
//
// Both callbacks will be GUARANTEED to execute before the function return.
func New(name string, decoder func(io.Reader) error, onChange func(error)) (res *Config, err error) {
	once := new(sync.Once)
	res = new(Config)
	res.name = name
	res.stop = make(chan struct{})
	ready := make(chan error)
	defer close(ready)
	if res.watcher, err = fsnotify.NewWatcher(); err != nil {
		err = errors.Wrap(err, "creating fsnotify watcher")
		return
	}
	if err = res.watcher.Add(res.name); err != nil {
		err = errors.Wrap(err, "adding filename to watcher")
		return
	}
	go func() {
		defer res.watcher.Close()
		for {
			if changed, err := res.read(decoder); changed || err != nil {
				onChange(err)
				once.Do(func() {
					ready <- err
				})
			}
			select {
			case <-res.stop:
				return
			case <-res.watcher.Events:
				time.Sleep(time.Second * 1) // Let the file settle for a while
			case <-time.After(time.Second * 30):
			}
		}
	}()
	err = <-ready
	return
}

// Close stops periodic reload loop
func (cfg *Config) Close() {
	close(cfg.stop)
}

func (cfg *Config) read(decoder func(io.Reader) error) (changed bool, err error) {
	fp, err := os.Open(cfg.name)
	if err != nil {
		err = errors.Wrap(err, "opening file")
		return
	}
	defer fp.Close()
	checksum := sha256.New()
	rdr := io.TeeReader(fp, checksum)
	err = decoder(rdr)
	changed = cfg.hash == nil || !bytes.Equal(cfg.hash.Sum(nil), checksum.Sum(nil))
	cfg.hash = checksum
	return
}
