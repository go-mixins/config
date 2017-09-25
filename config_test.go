package config_test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-mixins/config"
)

type testConfig struct {
	Users []string `json:"users"`
}

var testDir string

func copyFile(src string) (err error) {
	var srcFile, destFile *os.File
	if srcFile, err = os.Open(filepath.Join("testdata", src)); err != nil {
		return
	}
	defer srcFile.Close()
	if destFile, err = os.Create(filepath.Join(testDir, src)); err != nil {
		return
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, srcFile)
	return
}

func testMain(m *testing.M) int {
	var err error
	if testDir, err = ioutil.TempDir("", "configtest"); err != nil {
		panic(err)
	}
	defer os.RemoveAll(testDir)
	if err = copyFile("testcfg.json"); err != nil {
		panic(err)
	}
	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func wait(t *testing.T, event chan bool) {
	select {
	case <-event:
		break
	case <-time.After(time.Second * 1):
		t.Fatal("reload timed out")
	}
}

func TestNew(t *testing.T) {
	var c *testConfig
	var mu sync.RWMutex
	loaded := make(chan bool, 1)
	defer close(loaded)

	cfgName := filepath.Join(testDir, "testcfg.json")
	cfg, err := config.New(cfgName, func(r io.Reader) error {
		mu.RLock()
		defer mu.RUnlock()
		return json.NewDecoder(r).Decode(&c)
	}, func(changed bool, err error) {
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
		t.Logf("triggered: %#v", c)
		if changed {
			loaded <- true
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	wait(t, loaded)
	if len(c.Users) != 2 {
		t.Fatalf("invalid data loaded: %+v", c)
	}
	func() {
		mu.Lock()
		defer mu.Unlock()
		c.Users = append(c.Users, "Larry")
		fp, err := os.Create(cfgName)
		if err != nil {
			t.Fatal(err)
		}
		defer fp.Close()
		if err = json.NewEncoder(fp).Encode(c); err != nil {
			t.Fatal(err)
		}
	}()
	wait(t, loaded)
}
