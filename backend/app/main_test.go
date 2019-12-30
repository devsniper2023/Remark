package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Main(t *testing.T) {

	dir, err := ioutil.TempDir(os.TempDir(), "remark42")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	port := chooseRandomUnusedPort()
	os.Args = []string{"test", "server", "--secret=123456", "--store.bolt.path=" + dir, "--backup=/tmp",
		"--avatar.fs.path=" + dir, "--port=" + strconv.Itoa(port), "--url=https://demo.remark42.com", "--dbg", "--notify.type=none"}

	done := make(chan struct{})
	go func() {
		<-done
		e := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, e)
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		main()
		wg.Done()
	}()

	waitForHTTPServerStart(port)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))

	close(done)
	wg.Wait()
}

func TestGetDump(t *testing.T) {
	dump := getDump()
	assert.True(t, strings.Contains(dump, "goroutine"))
	assert.True(t, strings.Contains(dump, "[running]"))
	assert.True(t, strings.Contains(dump, "backend/app/main.go"))
	log.Printf("\n dump: %s", dump)
}

func chooseRandomUnusedPort() (port int) {
	for i := 0; i < 10; i++ {
		port = 40000 + int(rand.Int31n(10000))
		if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err == nil {
			_ = ln.Close()
			break
		}
	}
	return port
}

func waitForHTTPServerStart(port int) {
	// wait for up to 5 seconds for server to start before returning it
	client := http.Client{Timeout: time.Second}
	for i := 0; i < 500; i++ {
		time.Sleep(time.Millisecond * 10)
		if resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port)); err == nil {
			_ = resp.Body.Close()
			return
		}
	}
}
