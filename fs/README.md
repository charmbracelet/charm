# Charm FS

## Example

```go
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"ioutil"
	"os"

	charmfs "github.com/charmbracelet/charm/fs"
)

func main() {
	// Open the file system
	cfs, err := charmfs.NewFS()
	if err != nil {
		panic(err)
	}
	// Write a file
	data := []byte("some data")
	err = ioutil.WriteFile("/tmp/data", data, 0644)
	if err != nil {
		panic(err)
	}
	file, err := os.Open("/tmp/data")
	if err != nil {
		panic(err)
	}
	err = cfs.WriteFile("/our/test/data", file)
	if err != nil {
		panic(err)
	}
	// Get a file
	f, err := cfs.Open("/our/test/data")
	if err != nil {
		panic(err)
	}
	buf = bytes.NewBuffer(nil)
	_, err = io.Copy(buf, f)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(buf.Bytes()))

	// Or use fs.ReadFileFS
	bs, err := cfs.ReadFile("/our/test/data")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(bs))

	// Since we're using fs.FS interfaces we can also do things like walk a tree
	err = fs.WalkDir(cfs, "/", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path)
		return nil
	})
	if err != nil {
		panic(err)
	}
}
```
