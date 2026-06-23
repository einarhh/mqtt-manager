package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounce coalesces bursty filesystem events (an atomic save fires several)
// into a single onChange call.
const watchDebounce = 250 * time.Millisecond

// watchable reports whether a filename is one the app cares about.
func watchable(name string) bool {
	return strings.HasSuffix(name, ".js") || name == "index.json"
}

// Watch invokes onChange (debounced) whenever the plugins directory or its
// plugin files change on disk — so edits made outside the app are picked up
// without a manual reload. It returns a stop function; call it to tear down.
func (s *Store) Watch(onChange func()) (func(), error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.Add(s.dir); err != nil {
		_ = w.Close()
		return nil, err
	}
	// Also watch existing files directly so in-place edits (not just the
	// temp-file+rename that editors and our own writes use) are detected.
	if entries, err := os.ReadDir(s.dir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && watchable(e.Name()) {
				_ = w.Add(filepath.Join(s.dir, e.Name()))
			}
		}
	}

	stop := make(chan struct{})
	go func() {
		var timer *time.Timer
		fire := func() {
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(watchDebounce, onChange)
		}
		for {
			select {
			case <-stop:
				_ = w.Close()
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				name := filepath.Base(ev.Name)
				if strings.HasSuffix(name, ".tmp") || !watchable(name) {
					continue
				}
				// Watch newly-created files so future in-place edits fire too.
				if ev.Op&fsnotify.Create != 0 {
					_ = w.Add(ev.Name)
				}
				fire()
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return func() { close(stop) }, nil
}
