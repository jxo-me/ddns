package config

import (
	"github.com/jxo-me/ddns/pkg/watcher"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type mockNotifier struct {
	configs []Root
}

func (n *mockNotifier) ConfigDidUpdate(c Root) {
	n.configs = append(n.configs, c)
}

type mockFileWatcher struct {
	path     string
	notifier watcher.Notification
	ready    chan struct{}
}

func (w *mockFileWatcher) Start(n watcher.Notification) {
	w.notifier = n
	w.ready <- struct{}{}
}

func (w *mockFileWatcher) Add(string) error {
	return nil
}

func (w *mockFileWatcher) Shutdown() {

}

func (w *mockFileWatcher) TriggerChange() {
	w.notifier.WatcherItemDidChange(w.path)
}

func TestConfigChanged(t *testing.T) {
	filePath := "config.yaml"
	f, err := os.Create(filePath)
	assert.NoError(t, err)
	defer func() {
		_ = f.Close()
		_ = os.Remove(filePath)
	}()
	c := &Root{
		DDns: []DDnsConfig{
			{
				Name:  "alidns",
				TTL:   "600",
				Delay: 600,
			},
		},
	}
	configRead := func(configPath string, log *zerolog.Logger) (Root, error) {
		return *c, nil
	}
	wait := make(chan struct{})
	w := &mockFileWatcher{path: filePath, ready: wait}

	log := zerolog.Nop()

	service, err := NewFileManager(w, filePath, &log)
	service.ReadConfig = configRead
	assert.NoError(t, err)

	n := &mockNotifier{}
	go service.Start(n)

	<-wait
	c.DDns = append(c.DDns, DDnsConfig{Name: "cloudflare", TTL: "500"})
	w.TriggerChange()

	service.Shutdown()

	assert.Len(t, n.configs, 2, "did not get 2 config updates as expected")
	assert.Len(t, n.configs[0].DDns, 1, "not the amount of forwarders expected")
	assert.Len(t, n.configs[1].DDns, 2, "not the amount of forwarders expected")

	assert.Equal(t, n.configs[0].DDns[0].Name, c.DDns[0].Name, "DDns name don't match")
	assert.Equal(t, n.configs[1].DDns[0].Name, c.DDns[0].Name, "DDns name don't match")
	assert.Equal(t, n.configs[1].DDns[1].Name, c.DDns[1].Name, "DDns name don't match")
}
