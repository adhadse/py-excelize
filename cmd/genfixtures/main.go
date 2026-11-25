// Command genfixtures regenerates testdata/*.xlsx used by the Python tests.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/adhadse/py-excelize/internal/fixtures"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("genfixtures", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("out", "testdata", "output directory")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	paths, err := fixtures.WriteAll(*dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintln(stdout, "wrote", paths[name])
	}
	return 0
}
