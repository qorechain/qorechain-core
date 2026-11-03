package govauth

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
//
// The sweep used to look for the spelling `authority` only, so the bridge's
// `admin` messages and every licence- or registry-gated role were outside it
// (QSR-2026-0609). Now EVERY signer of every routable message is classified:
//
//   - privileged by name (privilegedSignerGuards): the handler must reach the
//     guard named for that signer;
//   - role-gated (roleGated): a role the chain grants by licence or registry,
//     named message by message with the check its handler must reach;
//   - the acting user (userSignerNames, userRoleMessages): no privilege, the
//     signer acts on its own funds or on objects it owns, named one by one;
//
// and a signer the test has never seen fails it, whatever its spelling.

var (
	reMsgHeader      = regexp.MustCompile(`(?m)^message\s+(Msg\w+)\s*\{`)
	reSignerOption   = regexp.MustCompile(`cosmos\.msg\.v1\.signer\)\s*=\s*"(\w+)"`)
	reMsgServerIface = regexp.MustCompile(`(?s)type MsgServer interface \{(.*?)\n\}`)
	reLineComment    = regexp.MustCompile(`//[^\n]*`)
)

// privilegedSignerGuards maps a signer field name that carries an
// administrative role to the guard every handler of such a message must reach.
// By name, so a new message using the same spelling is swept automatically.
var privilegedSignerGuards = map[string]string{
	"authority": "govauth.Assert",
	// The bridge admin is a keeper-held address or a qcb_bridge licence
	// holder; governance (the authority) also passes where the handler
	// allows it.
	"admin": "requireBridgeAdmin",
}

// roleCheck is the check a role-gated message's handler must reach, and why.
type roleCheck struct {
	guard string
	why   string
}

// roleGated names, one by one, the messages whose signer is a role the chain
// grants by licence or registry rather than by governance.
var roleGated = map[string]roleCheck{
	"bridge/MsgRegisterBridgeValidator": {"HasActiveLicense", "the operator must hold the qcb_bridge or a validator_<chain> licence"},
	"bridge/MsgBridgeAttestation":       {"GetBridgeValidator", "the attester must be a registered bridge validator"},
}

// userSignerNames are signer spellings that only ever mean "the account acting
// for itself".
var userSignerNames = map[string]bool{"sender": true}

// userRoleMessages names, one by one, the messages whose signer has another
// spelling but is still an ordinary account acting on its own funds or on an
// object it created. Each is a decision, not a pattern.
var userRoleMessages = map[string]string{
	"abstractaccount/MsgCreateAbstractAccount": "owner creates its own abstract account",
	"abstractaccount/MsgUpdateSpendingRules":   "owner of the account; the keeper checks ownership",
	"abstractaccount/MsgRegisterAuthenticator": "owner of the account; the keeper checks ownership",
	"abstractaccount/MsgRevokeAuthenticator":   "owner of the account; the keeper checks ownership",
	"abstractaccount/MsgExecuteEVM":            "relayer submits; the account's authenticator signature authorises the action",
	"abstractaccount/MsgExecuteCosmos":         "relayer submits; the account's authenticator signature authorises the action",
	"amm/MsgCreatePool":                        "any account creates a pool with its own liquidity",
	"bridge/MsgUpdateEthLightClient":           "permissionless relay; the update is verified against the sync committee",
	"lightnode/MsgRegisterLightNode":           "operator registers its own node",
	"lightnode/MsgHeartbeat":                   "operator of the node; the keeper checks it",
	"lightnode/MsgDeregisterLightNode":         "operator of the node; the keeper checks it",
	"lightnode/MsgClaimLightNodeRewards":       "operator claims its own rewards",
	"multilayer/MsgRegisterSidechain":          "any account registers a layer with its own stake",
	"multilayer/MsgRegisterPaychain":           "any account registers a layer with its own stake",
	"multilayer/MsgAnchorState":                "relayer; the anchor is verified against the layer creator's registered key",
	"multilayer/MsgChallengeAnchor":            "any account challenges with its own bond",
	"rdk/MsgCreateRollup":                      "any account creates a rollup with its own bond",
	"rdk/MsgPauseRollup":                       "creator of the rollup; the keeper checks it",
	"rdk/MsgResumeRollup":                      "creator of the rollup; the keeper checks it",
	"rdk/MsgStopRollup":                        "creator of the rollup; the keeper checks it",
	"rdk/MsgSetVerificationKey":                "creator of the rollup, before its first batch; the keeper checks it",
	"rdk/MsgSubmitBatch":                       "sequencer of the rollup; the keeper checks it",
	"rdk/MsgChallengeBatch":                    "any account challenges with its own bond",
	"rdk/MsgResolveChallenge":                  "the outcome is computed from the stored proof; anyone may trigger it from rdk-challenge-settlement",
	"rdk/MsgExecuteWithdrawal":                 "submitter; the withdrawal is proven against the finalised batch",
	"stakingtier/MsgLockDelegation":            "delegator locks its own delegation",
	"stakingtier/MsgUnlockEarly":               "delegator unlocks its own delegation",
	"stakingtier/MsgClaimBonus":                "holder claims its own bonus",
}

// Deliberate exceptions to a privileged guard, named one by one so each is a
// decision rather than a pattern that might quietly absorb a real gap.
var gatedElsewhere = map[string]string{
	"license/MsgGrantLicense":              "keeper checks the store-backed grant authority",
	"license/MsgRevokeLicense":             "keeper checks the store-backed grant authority",
	"license/MsgSuspendLicense":            "keeper checks the store-backed grant authority",
	"license/MsgResumeLicense":             "keeper checks the store-backed grant authority",
	"inflation/MsgUpdateParams":            "keeper checks k.Authority(), the same derived gov address",
	"burn/MsgReleaseStrandedRewards":       "handler compares msg.Authority with the gov module address the app passes to NewBurnKeeper",
	"lightnode/MsgReleaseLightNodeReserve": "handler compares msg.Authority with k.Authority(), the gov module address fixed in the keeper constructor",
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

// protoMessages returns each top-level Msg message of a proto file with its
// body, matching braces so nested options, oneofs and comments containing a
// brace cannot cut a body short (the earlier `[^}]*?` scan stopped at the
// first one).
func protoMessages(src string) map[string]string {
	src = reLineComment.ReplaceAllString(src, "")
	out := map[string]string{}
	for _, loc := range reMsgHeader.FindAllStringSubmatchIndex(src, -1) {
		name := src[loc[2]:loc[3]]
		depth, end := 0, -1
		for j := loc[1] - 1; j < len(src); j++ {
			if src[j] == '{' {
				depth++
			} else if src[j] == '}' {
				depth--
				if depth == 0 {
					end = j
					break
				}
			}
		}
		if end > 0 {
			out[name] = src[loc[1]:end]
		}
	}
	return out
}

// sweep is what the proto and generated-code scan finds in this repository.
type sweep struct {
	signers  map[string][]string // module/Msg -> declared signer fields
	routable map[string]bool     // module/Msg present in its module's MsgServer
}

func scanDeclarations(t *testing.T, core string) sweep {
	t.Helper()
	s := sweep{signers: map[string][]string{}, routable: map[string]bool{}}
	collect := func(p, body string) {
		if strings.HasSuffix(filepath.Base(p), "_test.proto") {
			return
		}
		mod := moduleOf(p)
		for name, msgBody := range protoMessages(body) {
			if strings.HasSuffix(name, "Response") {
				continue
			}
			key := mod + "/" + name
			var fields []string
			for _, m := range reSignerOption.FindAllStringSubmatch(msgBody, -1) {
				fields = append(fields, m[1])
			}
			if prev, ok := s.signers[key]; ok && strings.Join(prev, ",") != strings.Join(fields, ",") {
				t.Fatalf("%s is declared twice with different signers (%v, %v)", key, prev, fields)
			}
			s.signers[key] = fields
		}
	}
	walkFiles(t, filepath.Join(core, "proto"), ".proto", collect)
	walkFiles(t, filepath.Join(core, "x"), ".proto", collect)
	if len(s.signers) == 0 {
		t.Fatal("found no Msg messages at all; the scan is broken, not the code")
	}

	// A message absent from the generated MsgServer interface has no route
	// (x/multilayer's MsgUpdateParams is exactly this) and cannot be sent.
	walkFiles(t, filepath.Join(core, "x"), "tx.pb.go", func(p, body string) {
		mod := moduleOf(p)
		for _, iface := range reMsgServerIface.FindAllStringSubmatch(body, -1) {
			for key := range s.signers {
				m, name, _ := strings.Cut(key, "/")
				if m == mod && strings.Contains(iface[1], "*"+name+")") {
					s.routable[key] = true
				}
			}
		}
	})
	if len(s.routable) == 0 {
		t.Fatal("no message appears routable; the scan is broken")
	}
	return s
}

// classification is what one routable message's signer means.
type classification struct {
	guard string // the check its handlers must reach; "" for a user role
	why   string
}

// classify returns how the sweep treats key, or an error text for a signer it
// does not know.
func classify(key string, fields []string) (classification, string) {
	if len(fields) == 0 {
		return classification{}, "declares no cosmos.msg.v1.signer"
	}
	if len(fields) > 1 {
		return classification{}, "declares more than one signer: " + strings.Join(fields, ", ")
	}
	f := fields[0]
	if guard, ok := privilegedSignerGuards[f]; ok {
		if why, ok := gatedElsewhere[key]; ok {
			return classification{why: why}, ""
		}
		return classification{guard: guard, why: "signer " + f}, ""
	}
	if rc, ok := roleGated[key]; ok {
		return classification{guard: rc.guard, why: rc.why}, ""
	}
	if userSignerNames[f] {
		return classification{why: "the acting account"}, ""
	}
	if why, ok := userRoleMessages[key]; ok {
		return classification{why: why}, ""
	}
	return classification{}, `signer "` + f + `" is neither a known privileged role nor a listed user role`
}

// TestEverySignerIsClassified needs no handler: it reads the proto files and
// the generated MsgServer interfaces only, so it runs in every checkout,
// public CI included, and fails on a signer nobody has classified.
func TestEverySignerIsClassified(t *testing.T) {
	core, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	s := scanDeclarations(t, core)

	var unknown []string
	privileged := 0
	for key := range s.routable {
		c, problem := classify(key, s.signers[key])
		if problem != "" {
			unknown = append(unknown, key+": "+problem)
		}
		if c.guard != "" {
			privileged++
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		t.Fatalf("routable messages whose signer the sweep cannot classify (add a guard, a roleGated entry or a userRoleMessages entry, each with its reason):\n  %s",
			strings.Join(unknown, "\n  "))
	}

	// Stale entries hide mistakes: a listed message must exist and be routable.
	for _, list := range []map[string]string{userRoleMessages, gatedElsewhere} {
		for key := range list {
			if !s.routable[key] {
				t.Fatalf("%s is listed but is not a routable message (renamed or removed?)", key)
			}
		}
	}
	for key := range roleGated {
		if !s.routable[key] {
			t.Fatalf("%s is listed in roleGated but is not a routable message", key)
		}
	}
	t.Logf("%d messages declared, %d routable, %d privileged or role-gated", len(s.signers), len(s.routable), privileged)
}

// handler is one function that takes a Msg, by pointer (an entry point, as
// every MsgServer method is) or by value (a helper the entry point delegates
// to, such as the bridge's Handle* functions).
type handler struct {
	module  string
	fn      string
	msgs    []string
	body    string
	pointer bool
}

func TestEveryRoutablePrivilegedMessageIsGated(t *testing.T) {
	core, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	// Some handlers live in an overlay tree outside this module, so the roots to
	// scan are supplied by the build rather than hardcoded here. Without it this
	// test can only see what is in this repository, and a partial sweep that
	// reports success is worse than one that says it did not run. In CI that
	// is a failure, unless the workflow says outright that it runs the
	// declarations half only (the public build, which has no handlers to see;
	// TestEverySignerIsClassified still runs there).
	extra := filepath.SplitList(os.Getenv("QORE_HANDLER_ROOTS"))
	if len(extra) == 0 || extra[0] == "" {
		if os.Getenv("CI") != "" && os.Getenv("QORE_AUTHORITY_SWEEP") != "declarations-only" {
			t.Fatal("QORE_HANDLER_ROOTS is not set in CI: the handler sweep cannot see the handlers. " +
				"Run it from the full build (make test in the proprietary tree), or set " +
				"QORE_AUTHORITY_SWEEP=declarations-only where no handler tree exists")
		}
		t.Skip("QORE_HANDLER_ROOTS not set; run this from the full build so every handler is visible")
	}

	for _, root := range extra {
		if st, err := os.Stat(filepath.Join(root, "x")); err != nil || !st.IsDir() {
			t.Fatalf("QORE_HANDLER_ROOTS entry %q has no x/ tree (a relative path is read from this package's directory)", root)
		}
	}

	s := scanDeclarations(t, core)
	required := map[string]classification{}
	for key := range s.routable {
		c, problem := classify(key, s.signers[key])
		if problem != "" {
			t.Fatalf("%s: %s (TestEverySignerIsClassified names every such message)", key, problem)
		}
		if c.guard != "" {
			required[key] = c
		}
	}

	// Every function taking a Msg, with its own body.
	var handlers []handler
	parse := func(p, src string) {
		// Generated protobuf code defines Marshal/Unmarshal/Size over the very
		// same Msg pointers; those are not handlers and must not be demanded to
		// carry a guard. Tests are not handlers either.
		if strings.HasSuffix(p, ".pb.go") || strings.HasSuffix(p, ".pb.gw.go") || strings.HasSuffix(p, "_test.go") {
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
			pointer := false
			if fd.Type.Params != nil {
				for _, param := range fd.Type.Params.List {
					expr := param.Type
					isPtr := false
					if star, ok := expr.(*ast.StarExpr); ok {
						expr, isPtr = star.X, true
					}
					var ident *ast.Ident
					switch x := expr.(type) {
					case *ast.SelectorExpr:
						ident = x.Sel
					case *ast.Ident:
						ident = x
					}
					if ident != nil && strings.HasPrefix(ident.Name, "Msg") {
						msgs = append(msgs, ident.Name)
						pointer = pointer || isPtr
					}
				}
			}
			if len(msgs) == 0 {
				continue
			}
			handlers = append(handlers, handler{
				module:  mod,
				fn:      fd.Name.Name,
				msgs:    msgs,
				body:    src[fset.Position(fd.Body.Pos()).Offset:fset.Position(fd.Body.End()).Offset],
				pointer: pointer,
			})
		}
	}
	walkFiles(t, filepath.Join(core, "x"), ".go", parse)
	for _, root := range extra {
		walkFiles(t, filepath.Join(root, "x"), ".go", parse)
	}

	// A function reaches a guard if its own body calls it, or if it delegates
	// to a function in the same module that reaches it (x/pqc's wrappers call
	// *Impl, the bridge's MsgServer calls Handle*). Iterate to a fixed point
	// so a chain of any depth resolves. One fixed point per guard: reaching
	// govauth.Assert says nothing about a licence check.
	reaches := func(guard string) map[string]bool {
		got := map[string]bool{}
		for _, h := range handlers {
			if strings.Contains(h.body, guard+"(") {
				got[h.module+"."+h.fn] = true
			}
		}
		for changed := true; changed; {
			changed = false
			for _, h := range handlers {
				key := h.module + "." + h.fn
				if got[key] {
					continue
				}
				for g := range got {
					m, name, _ := strings.Cut(g, ".")
					if m == h.module && strings.Contains(h.body, name+"(") {
						got[key] = true
						changed = true
						break
					}
				}
			}
		}
		return got
	}
	byGuard := map[string]map[string]bool{}

	// EVERY entry point for a required message must reach its guard, and every
	// required message must have at least one entry point the sweep saw: a
	// message whose handler the sweep cannot find is not vouched for.
	var ungated []string
	seen := map[string]bool{}
	for _, h := range handlers {
		if !h.pointer {
			continue
		}
		for _, msg := range h.msgs {
			key := h.module + "/" + msg
			c, ok := required[key]
			if !ok {
				continue
			}
			seen[key] = true
			if byGuard[c.guard] == nil {
				byGuard[c.guard] = reaches(c.guard)
			}
			if !byGuard[c.guard][h.module+"."+h.fn] {
				ungated = append(ungated, key+" via "+h.module+"."+h.fn+"() does not reach "+c.guard+" ("+c.why+")")
			}
		}
	}
	for key, c := range required {
		if !seen[key] {
			ungated = append(ungated, key+": no handler found in the scanned trees ("+c.why+")")
		}
	}
	sort.Strings(ungated)
	if len(ungated) > 0 {
		t.Fatalf("routable privileged messages reachable through an unguarded handler:\n  %s",
			strings.Join(ungated, "\n  "))
	}
	t.Logf("%d routable messages, %d privileged or role-gated, %d handlers checked",
		len(s.routable), len(required), len(handlers))
}

// The classifier refuses what it has not been told about, whatever the
// spelling, and keeps the privileged names privileged.
func TestClassifyRefusesUnknownSigners(t *testing.T) {
	for _, tc := range []struct {
		key    string
		fields []string
	}{
		{"bridge/MsgNewThing", []string{"operator_admin"}},
		{"rdk/MsgNewThing", []string{"creator"}},
		{"pqc/MsgNoSigner", nil},
		{"pqc/MsgTwoSigners", []string{"sender", "authority"}},
	} {
		if _, problem := classify(tc.key, tc.fields); problem == "" {
			t.Fatalf("%s %v was classified", tc.key, tc.fields)
		}
	}
	if c, _ := classify("pqc/MsgAnything", []string{"authority"}); c.guard != "govauth.Assert" {
		t.Fatal("an authority signer is not held to govauth.Assert")
	}
	if c, _ := classify("bridge/MsgAnything", []string{"admin"}); c.guard != "requireBridgeAdmin" {
		t.Fatal("an admin signer is not held to requireBridgeAdmin")
	}
	if c, _ := classify("license/MsgGrantLicense", []string{"authority"}); c.guard != "" {
		t.Fatal("a named exception is still demanded the generic guard")
	}
}
