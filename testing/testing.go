//coverage:ignore file

package testing

import (
	"os"
	"path"
	"runtime"
)

// Workaround for the problem in go tests where the current working directory is that of the
// code being executed rather than the project root.
// See https://brandur.org/fragments/testing-go-project-root.
func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}
