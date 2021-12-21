package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Thread safe buffer to avoid data races when setting a custom writer
// for the log
type Buffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}

func (b *Buffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.String()
}

func TestServe(t *testing.T) {
	tempDir := t.TempDir()
	buf := Buffer{}
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	ServeCmd.SetArgs([]string{"--data-dir", tempDir})

	ctx, cancel := context.WithCancel(context.Background())
	go ServeCmd.ExecuteContext(ctx)

	if !waitForServer(":35354") {
		assert.FailNow(t, "server did not start")
	}
	cancel()

	assert.DirExists(t, filepath.Join(tempDir, "db"))
	assert.Regexp(t, regexp.MustCompile("HTTP server listening on: :35354"), buf.String())
	// helps with debugging if test fails
	fmt.Println(buf.String())
}

func waitForServer(addr string) bool {
	for i := 0; i < 40; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}

	return false
}
