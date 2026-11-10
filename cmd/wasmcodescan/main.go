// Command wasmcodescan runs the x/wasmfilter code check over CosmWasm code
// files and lists which ones the wasm-code-filter protocol feature would
// refuse. It only reads files; it never talks to a node.
//
// Inputs are files or directories (scanned one level deep, every regular
// file). Gzip-wrapped files are unpacked as wasmd does for MsgStoreCode. Use
// it on:
//
//   - a copy of a node's code directory, <home>/wasm/wasm/state/wasm (one file
//     per stored code, named by checksum);
//   - files downloaded with `qorechaind query wasm code <id> <file>`;
//   - contract builds before they go into a store proposal.
//
// With -codes FILE (the JSON of `qorechaind query wasm list-code -o json`,
// all pages concatenated or one page) each checksum is mapped to its code ids.
//
// Output: one tab-separated line per file (verdict, checksum, size, code ids,
// functions, max br_table, type-index blocks, parameterised blocks, path,
// reason). Exit status 1 when any file fails, 2 on a usage or read error.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CosmWasm/wasmd/x/wasm/ioutils"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/qorechain/qorechain-core/x/wasmfilter"
)

type listCode struct {
	CodeInfos []struct {
		CodeID   string `json:"code_id"`
		DataHash string `json:"data_hash"`
	} `json:"code_infos"`
}

func main() {
	codesFile := flag.String("codes", "", "JSON output of `query wasm list-code -o json` (optional; several JSON documents may be concatenated)")
	failedOnly := flag.Bool("failed", false, "print only files that fail")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: wasmcodescan [-codes list.json] [-failed] FILE|DIR...\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	ids := map[string][]string{}
	if *codesFile != "" {
		bz, err := os.ReadFile(*codesFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		dec := json.NewDecoder(bytes.NewReader(bz))
		for {
			var page listCode
			if err := dec.Decode(&page); err == io.EOF {
				break
			} else if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", *codesFile, err)
				os.Exit(2)
			}
			for _, c := range page.CodeInfos {
				h := strings.ToLower(c.DataHash)
				ids[h] = append(ids[h], c.CodeID)
			}
		}
	}

	var files []string
	for _, arg := range flag.Args() {
		fi, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if !fi.IsDir() {
			files = append(files, arg)
			continue
		}
		entries, err := os.ReadDir(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		for _, e := range entries {
			if e.Type().IsRegular() {
				files = append(files, filepath.Join(arg, e.Name()))
			}
		}
	}
	sort.Strings(files)

	fmt.Println("verdict\tchecksum\tsize\tcode_ids\tfunctions\tmax_br_table\ttyped_blocks\tparam_blocks\tpath\treason")
	failed, seen := 0, map[string]bool{}
	for _, path := range files {
		bz, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if ioutils.IsGzip(bz) {
			raw, err := ioutils.Uncompress(bz, int64(wasmtypes.MaxWasmSize))
			if err != nil {
				failed++
				fmt.Printf("FAIL\t-\t%d\t-\t-\t-\t-\t-\t%s\tgzip: %v\n", len(bz), path, err)
				continue
			}
			bz = raw
		}
		sum := sha256.Sum256(bz)
		h := hex.EncodeToString(sum[:])
		st, verr := wasmfilter.Inspect(bz)
		verdict, reason := "pass", ""
		if verr != nil {
			verdict, reason = "FAIL", verr.Error()
			failed++
		}
		seen[h] = true
		if *failedOnly && verr == nil {
			continue
		}
		codeIDs := strings.Join(ids[h], ",")
		if codeIDs == "" {
			codeIDs = "-"
		}
		fmt.Printf("%s\t%s\t%d\t%s\t%d\t%d\t%d\t%d\t%s\t%s\n",
			verdict, h, len(bz), codeIDs, st.Functions, st.MaxBrTable, st.TypedBlocks, st.ParamBlocks, path, reason)
	}
	if len(ids) > 0 {
		var missing []string
		for h, c := range ids {
			if !seen[h] {
				missing = append(missing, fmt.Sprintf("%s (code %s)", h, strings.Join(c, ",")))
			}
		}
		sort.Strings(missing)
		for _, m := range missing {
			fmt.Fprintf(os.Stderr, "no file for %s\n", m)
		}
	}
	fmt.Fprintf(os.Stderr, "%d files, %d fail\n", len(files), failed)
	if failed > 0 {
		os.Exit(1)
	}
}
