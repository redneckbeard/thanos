package compiler

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/redneckbeard/thanos/parser"
)

var filename, label string

func init() {
	flag.StringVar(&filename, "filename", "", "name of the file to test compilation for")
	flag.StringVar(&label, "label", "", "label for the compilation test")
}

func TestCompile(t *testing.T) {
	rubyFiles, _ := filepath.Glob("testdata/ruby/*.rb")
	for _, ruby := range rubyFiles {
		base := filepath.Base(ruby)
		noExt := strings.TrimSuffix(base, filepath.Ext(base))
		if filename == "" || filename == noExt {
			goTgt := fmt.Sprintf("testdata/go/%s.go", noExt)
			program, err := parser.ParseFile(ruby)
			if err != nil {
				t.Error("Error parsing "+ruby+": ", err)
				continue
			}
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recovered test failure in %s: %s\n\n%s", ruby, r, string(debug.Stack()))
				}
			}()
			result, err := Compile(program)
			if err != nil {
				t.Error(err)
				continue
			}
			for path, translated := range result.Files {
				var expectedPath string
				if path == "main.go" {
					expectedPath = goTgt
				} else {
					// Module package files: testdata/go/<name>/<path>
					expectedPath = fmt.Sprintf("testdata/go/%s/%s", noExt, path)
				}
				f, err := os.Open(expectedPath)
				if err != nil {
					t.Errorf("Missing expected output file %s for %s", expectedPath, ruby)
					continue
				}
				b, err := ioutil.ReadAll(f)
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
				if translated != string(b) {
					cmd := exec.Command("diff", "--color=always", "-b", "-c", expectedPath, "-")
					cmd.Stdin = strings.NewReader(translated)
					var out bytes.Buffer
					cmd.Stdout = &out
					cmd.Run()
					if len(strings.TrimSpace(out.String())) > 0 {
						t.Errorf("Got unexpected result translating %s (%s):\n\n%s\n", ruby, path, out.String())
					}
				}
			}
		}
	}
}
