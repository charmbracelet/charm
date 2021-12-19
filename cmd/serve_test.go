package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServe(t *testing.T) {
	tempDir := t.TempDir()
	out := bytes.NewBufferString("")
	log.SetOutput(out)
	ServeCmd.SetArgs([]string{"--data-dir", tempDir})

	ctx, cancel := context.WithCancel(context.Background())
	go ServeCmd.ExecuteContext(ctx)
	defer cancel()

	if !waitForServer(":35354") {
		assert.FailNow(t, "server did not start")
	}

	dbDir := filepath.Join(tempDir, "db")
	assert.Regexp(t, regexp.MustCompile("HTTP server listening on: :35354"), out.String())
	assert.Regexp(t, regexp.MustCompile(fmt.Sprintf("Opening SQLite db: %s", dbDir)), out.String())
}

func waitForServer(addr string) bool {
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}

	return false
}
