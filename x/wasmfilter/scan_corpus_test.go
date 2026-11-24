package wasmfilter

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScanCorpus reports, for a directory of contract codes
// (WASMFILTER_CORPUS, raw .wasm or gzip .wasm.gz files), how many pass the
// check and how many contain float, div/rem and memory operators. A tool for
// pricing decisions; it skips when the variable is not set.
func TestScanCorpus(t *testing.T) {
	dirs := os.Getenv("WASMFILTER_CORPUS")
	if dirs == "" {
		t.Skip("WASMFILTER_CORPUS not set")
	}
	var files []string
	for _, dir := range filepath.SplitList(dirs) {
		for _, pat := range []string{"*.wasm", "*.wasm.gz"} {
			m, err := filepath.Glob(filepath.Join(dir, pat))
			if err != nil {
				t.Fatal(err)
			}
			files = append(files, m...)
		}
	}
	var pass, refused, withFloat, withDiv, withMem, maxCustom int
	var maxCustomName string
	var floatNames []string
	for _, f := range files {
		bz, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(f, ".gz") {
			zr, err := gzip.NewReader(bytes.NewReader(bz))
			if err != nil {
				t.Fatal(err)
			}
			if bz, err = io.ReadAll(zr); err != nil {
				t.Fatal(err)
			}
		}
		st, err := Inspect(bz)
		if err != nil {
			refused++
			t.Logf("refused %s: %v", filepath.Base(f), err)
			continue
		}
		pass++
		if st.FloatOps > 0 {
			withFloat++
			floatNames = append(floatNames, filepath.Base(f))
		}
		if st.DivRemOps > 0 {
			withDiv++
		}
		if st.MemoryOps > 0 {
			withMem++
		}
		if st.CustomBytes > maxCustom {
			maxCustom, maxCustomName = st.CustomBytes, filepath.Base(f)
		}
	}
	t.Logf("%d codes: %d pass, %d refused; with float ops %d, with div/rem %d, with memory.size/grow %d",
		len(files), pass, refused, withFloat, withDiv, withMem)
	t.Logf("largest custom sections: %d bytes (%s)", maxCustom, maxCustomName)
	if len(floatNames) > 0 {
		t.Logf("codes with float ops: %s", strings.Join(floatNames, " "))
	}
}
