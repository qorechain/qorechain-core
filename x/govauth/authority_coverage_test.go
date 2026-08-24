package govauth

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A message whose declared signer is its own `authority` field is signed by
// whoever fills that field in. Such a message is safe ONLY if its handler calls
// govauth.Assert. This test finds every one of them mechanically, so a new admin
// message added months from now cannot ship ungated because the author did not
// know the rule.
//
// It exists because several such messages did ship ungated: x/pqc's
// MsgDisableAlgorithm alone could freeze the chain, and it survived a full first
// audit pass. A file-level version of this test was written first and PASSED
// with a guard deliberately deleted, because a sibling handler in the same file
// still mentioned govauth. Hence the AST: the guard must be in the handler's own
// call path, not merely somewhere in its file.

var (
	reAuthoritySigner = regexp.MustCompile(`(?m)^message\s+(Msg\w+)\s*\{[^}]*?cosmos\.msg\.v1\.signer\)\s*=\s*"authority"`)
	reMsgServerIface  = regexp.MustCompile(`(?s)type MsgServer interface \{(.*?)\n\}`)
)

// Deliberate exceptions, named one by one so each is a decision rather than a
// pattern that might quietly absorb a real gap.
var gatedElsewhere = map[string]string{
	"license/MsgGrantLicense":   "keeper checks the store-backed grant authority",
	"license/MsgRevokeLicense":  "keeper checks the store-backed grant authority",
	"license/MsgSuspendLicense": "keeper checks the store-backed grant authority",
	"license/MsgResumeLicense":  "keeper checks the store-backed grant authority",
	"inflation/MsgUpdateParams": "keeper checks k.Authority(), the same derived gov address",
}

// moduleOf names the module a path belongs to, from either
// ".../proto/qorechain/<mod>/..." or ".../x/<mod>/...". Both layouts exist;
// x/multilayer even carries its own proto tree alongside the shared one.
func moduleOf(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := len(parts) - 1; i > 0; i-- {
		if parts[i-1] == "qorechain" || parts[i-1] == "x" {
			return parts[i]
		}
	}
	return ""
}

func walkFiles(t *testing.T, root, suffix string, fn func(path, body string)) {
	t.Helper()
	if _, err := os.Stat(root); err != nil {
		return
	}
	if err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, suffix) {
			return nil
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		fn(p, string(b))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// handler is one function that takes a Msg pointer.
type handler struct {
	module string
	fn     string
	msgs   []string
	body   string
}

func TestEveryRoutableAuthorityMessageIsGated(t *testing.T) {
	core, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	// Some handlers live in an overlay tree outside this module, so the roots to
	// scan are supplied by the build rather than hardcoded here. Without it this
	// test can only see what is in this repository, and a partial sweep that
	// reports success is worse than one that says it did not run.
	extra := filepath.SplitList(os.Getenv("QORE_HANDLER_ROOTS"))
	if len(extra) == 0 || extra[0] == "" {
		t.Skip("QORE_HANDLER_ROOTS not set; run this from the full build so every handler is visible")
	}

	// 1. Every message declaring `authority` as its signer, keyed by module so
	//    two modules that both define MsgUpdateParams are never conflated.
	declared := map[string]string{}
	collect := func(p, body string) {
		for _, m := range reAuthoritySigner.FindAllStringSubmatch(body, -1) {
			declared[moduleOf(p)+"/"+m[1]] = p
		}
	}
	walkFiles(t, filepath.Join(core, "proto"), ".proto", collect)
	walkFiles(t, filepath.Join(core, "x"), ".proto", collect)
	if len(declared) == 0 {
		t.Fatal("found no authority-signed messages at all; the scan is broken, not the code")
	}

	// 2. Keep only those reachable as transactions. A message absent from the
	//    generated MsgServer interface has no route (x/multilayer's
	//    MsgUpdateParams is exactly this) and cannot be exploited.
	routable := map[string]bool{}
	walkFiles(t, filepath.Join(core, "x"), "tx.pb.go", func(p, body string) {
		mod := moduleOf(p)
		for _, iface := range reMsgServerIface.FindAllStringSubmatch(body, -1) {
			for key := range declared {
				m, name, _ := strings.Cut(key, "/")
				if m == mod && strings.Contains(iface[1], "*"+name+")") {
					routable[key] = true
				}
			}
		}
	})
	if len(routable) == 0 {
		t.Fatal("no authority-signed message appears routable; the scan is broken")
	}

	// 3. Collect every function taking a Msg pointer, with its own body.
	var handlers []handler
	parse := func(p, src string) {
		// Generated protobuf code defines Marshal/Unmarshal/Size over the very
		// same Msg pointers; those are not handlers and must not be demanded to
		// carry a guard. Tests are not handlers either.
		if strings.HasSuffix(p, ".pb.go") || strings.HasSuffix(p, "_test.go") {
			return
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, p, src, 0)
		if perr != nil {
			return // generated or build-tagged files we cannot parse are not handlers
		}
		mod := moduleOf(p)
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			// PARAMETERS only. Methods declared ON a Msg type (ValidateBasic,
			// GetSigners, the generated accessors) have it as their RECEIVER and
			// are not handlers; a text scan of the signature cannot tell them
			// apart, so read the AST.
			var msgs []string
			if fd.Type.Params != nil {
				for _, param := range fd.Type.Params.List {
					star, ok := param.Type.(*ast.StarExpr)
					if !ok {
						continue
					}
					var ident *ast.Ident
					switch x := star.X.(type) {
					case *ast.SelectorExpr:
						ident = x.Sel
					case *ast.Ident:
						ident = x
					}
					if ident != nil && strings.HasPrefix(ident.Name, "Msg") {
						msgs = append(msgs, ident.Name)
					}
				}
			}
			if len(msgs) == 0 {
				continue
			}
			handlers = append(handlers, handler{
				module: mod,
				fn:     fd.Name.Name,
				msgs:   msgs,
				body:   src[fset.Position(fd.Body.Pos()).Offset:fset.Position(fd.Body.End()).Offset],
			})
		}
	}
	walkFiles(t, filepath.Join(core, "x"), ".go", parse)
	for _, root := range extra {
		walkFiles(t, filepath.Join(root, "x"), ".go", parse)
	}

	// 4. A handler is guarded if its own body asserts, or if it delegates to a
	//    guarded function in the same module (x/pqc's wrappers call *Impl).
	//    Iterate to a fixed point so a chain of any depth resolves.
	guardedFn := map[string]bool{}
	for _, h := range handlers {
		if strings.Contains(h.body, "govauth.Assert") {
			guardedFn[h.module+"."+h.fn] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, h := range handlers {
			key := h.module + "." + h.fn
			if guardedFn[key] {
				continue
			}
			for g := range guardedFn {
				m, name, _ := strings.Cut(g, ".")
				if m == h.module && strings.Contains(h.body, name+"(") {
					guardedFn[key] = true
					changed = true
					break
				}
			}
		}
	}

	// 5. EVERY entry point for a routable message must be guarded.
	var ungated []string
	for _, h := range handlers {
		for _, msg := range h.msgs {
			key := h.module + "/" + msg
			if !routable[key] {
				continue
			}
			if _, ok := gatedElsewhere[key]; ok {
				continue
			}
			if !guardedFn[h.module+"."+h.fn] {
				ungated = append(ungated, key+" via "+h.module+"."+h.fn+"()")
			}
		}
	}
	if len(ungated) > 0 {
		t.Fatalf("routable authority-signed messages reachable through an unguarded handler:\n  %s",
			strings.Join(ungated, "\n  "))
	}
	t.Logf("%d authority-signed messages declared, %d routable, %d handlers checked",
		len(declared), len(routable), len(handlers))
}
