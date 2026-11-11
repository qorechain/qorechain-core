// Package wasmfilter is a deterministic, pure-Go check of CosmWasm bytecode
// that runs in front of the wasm VM. It walks the module's sections and every
// function body and refuses a narrow set of constructs that no contract built
// by the usual Rust toolchain contains, and anything it cannot fully parse.
//
// What it refuses (see Check):
//
//   - A block, loop or if whose block type is a type index (the multi-value
//     form). Contracts use the empty form (0x40) or a single value type, and the
//     CosmWasm static limits already allow at most one result per function, so
//     this is the only place a multi-value signature can appear. An `if` whose
//     type-index signature has parameters and that has no `else` is reported
//     with its own reason.
//   - A br_table with more than MaxBrTableTargets targets.
//   - Any opcode outside the feature set the CosmWasm validator enables (MVP,
//     sign extension, saturating float-to-int, mutable globals, multi-value).
//   - Any module it cannot parse: truncated or oversized sections, bad LEB128,
//     unbalanced control structure, unknown section ids or value types.
//
// The verdict depends on the bytes alone, so every node reaches the same one.
// The parser does not replace the VM's own validation; it only refuses more.
package wasmfilter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// MaxBrTableTargets is the most branch targets one br_table may list (the
// default target not counted). Contracts built with the CosmWasm optimizer use
// at most 84 (the 10 contract builds kept with the release, wasmd's and
// wasmvm's test contracts); the cap leaves a factor of about 50 above that.
const MaxBrTableTargets = 4096

// Reasons a module is refused. Check wraps one of these.
var (
	ErrMalformed          = errors.New("malformed or truncated wasm")
	ErrTypedBlock         = errors.New("block type with a type-index signature")
	ErrParamIfWithoutElse = errors.New("if with block parameters and no else")
	ErrBrTableTooLarge    = errors.New("br_table has too many targets")
	ErrUnsupportedOpcode  = errors.New("opcode outside the supported feature set")
	// ErrFloatOpcode is the refusal of a module that contains float
	// operators, from the wasm-float-ops-refused switch (see CheckFloats).
	ErrFloatOpcode = errors.New("float (f32/f64) operator")
)

// CheckFloats returns ErrFloatOpcode (wrapped, with the count) when a module
// that passes Check contains float operators, and nil otherwise. The VM
// meters every operator at one flat price, and float operators (with the
// VM's NaN canonicalisation) cost the validator up to thousands of times the
// CPU of the cheap ones per unit of gas; contracts built from the usual Rust
// toolchain contain none (no release contract, no contract stored on the
// testnet, none of wasmd's test contracts). A module Check refuses is left to
// Check.
func CheckFloats(code []byte) error {
	st, err := Inspect(code)
	if err != nil || st.FloatOps == 0 {
		return nil
	}
	return fmt.Errorf("%w: %d float operators", ErrFloatOpcode, st.FloatOps)
}

const (
	secCustom    = 0
	secType      = 1
	secCode      = 10
	secDataCount = 12

	opBlock = 0x02
	opLoop  = 0x03
	opIf    = 0x04
	opElse  = 0x05
	opEnd   = 0x0B
)

type funcType struct {
	params  uint32
	results uint32
}

type reader struct {
	b   []byte
	pos int
}

func (r *reader) eof() bool { return r.pos >= len(r.b) }

func (r *reader) byte() (byte, error) {
	if r.pos >= len(r.b) {
		return 0, ErrMalformed
	}
	c := r.b[r.pos]
	r.pos++
	return c, nil
}

func (r *reader) bytes(n int) ([]byte, error) {
	if n < 0 || n > len(r.b)-r.pos {
		return nil, ErrMalformed
	}
	out := r.b[r.pos : r.pos+n]
	r.pos += n
	return out, nil
}

// u32 reads an unsigned LEB128 of at most 5 bytes whose value fits 32 bits.
func (r *reader) u32() (uint32, error) {
	var v uint64
	for i := 0; i < 5; i++ {
		c, err := r.byte()
		if err != nil {
			return 0, err
		}
		v |= uint64(c&0x7f) << (7 * i)
		if c&0x80 == 0 {
			if v > math.MaxUint32 {
				return 0, ErrMalformed
			}
			return uint32(v), nil
		}
	}
	return 0, ErrMalformed
}

// signed reads a signed LEB128 of at most maxBytes bytes and checks that the
// value fits `bits` bits.
func (r *reader) signed(bits uint, maxBytes int) (int64, error) {
	var v int64
	var shift uint
	for i := 0; i < maxBytes; i++ {
		c, err := r.byte()
		if err != nil {
			return 0, err
		}
		if shift < 64 {
			v |= int64(c&0x7f) << shift
		}
		shift += 7
		if c&0x80 == 0 {
			if shift < 64 && c&0x40 != 0 {
				v |= -1 << shift
			}
			if bits < 64 {
				lo, hi := int64(-1)<<(bits-1), int64(1)<<(bits-1)-1
				if v < lo || v > hi {
					return 0, ErrMalformed
				}
			} else if shift > 64 {
				// 10th byte: only the sign bit and its extension may be set.
				if c != 0x00 && c != 0x7f {
					return 0, ErrMalformed
				}
			}
			return v, nil
		}
	}
	return 0, ErrMalformed
}

func isNumType(c byte) bool { return c >= 0x7C && c <= 0x7F } // f64, f32, i64, i32

// Stats describes what Inspect saw, up to the first refusal.
type Stats struct {
	Functions   int // function bodies walked
	MaxBrTable  int // largest br_table target count (default not counted)
	TypedBlocks int // block, loop and if with a type-index block type
	ParamBlocks int // of those, signatures with parameters
	FloatOps    int // operators that read, write, compute or convert f32/f64 values
	DivRemOps   int // integer div and rem operators
	MemoryOps   int // memory.size and memory.grow
	CustomBytes int // bytes in custom sections (names, debug info, producers)
	// Operators walked, and the count of each operator class
	// (OpClass), for the compiled-size estimate.
	Ops     int
	ByClass [NumOpClasses]int
}

// OpClass groups operators by how much compiled code one of them makes.
type OpClass int

const (
	OpOther        OpClass = iota // everything below not listed
	OpCall                        // call
	OpCallIndirect                // call_indirect
	OpBranch                      // block, loop, if, br_if, br_table
	OpFlow                        // else, end, br, return
	OpLoad                        // memory loads
	OpStore                       // memory stores
	OpDivRem                      // integer division and remainder
	OpBitCount                    // eqz, clz, ctz, popcnt
	OpMemory                      // memory.size, memory.grow
	NumOpClasses
)

// opClass is the class of a single-byte operator.
func opClass(op byte) OpClass {
	switch {
	case op == 0x10:
		return OpCall
	case op == 0x11:
		return OpCallIndirect
	case op == opBlock || op == opLoop || op == opIf || op == 0x0D || op == 0x0E:
		return OpBranch
	case op == opElse || op == opEnd || op == 0x0C || op == 0x0F:
		return OpFlow
	case op >= 0x28 && op <= 0x35:
		return OpLoad
	case op >= 0x36 && op <= 0x3E:
		return OpStore
	case isDivRemOp(op):
		return OpDivRem
	case op == 0x45 || op == 0x50 || (op >= 0x67 && op <= 0x69) || (op >= 0x79 && op <= 0x7B):
		return OpBitCount
	case op == 0x3F || op == 0x40:
		return OpMemory
	}
	return OpOther
}

// IsFloatOp reports whether the single-byte opcode op handles a float value:
// f32/f64 loads, stores and constants, comparisons, arithmetic and every
// conversion from or to a float type. The 0xFC saturating truncations are
// float operators too (see checkBody).
func IsFloatOp(op byte) bool {
	switch {
	case op == 0x2A || op == 0x2B || op == 0x38 || op == 0x39: // f32/f64 load, store
		return true
	case op == 0x43 || op == 0x44: // f32.const, f64.const
		return true
	case op >= 0x5B && op <= 0x66: // f32/f64 comparisons
		return true
	case op >= 0x8B && op <= 0xA6: // f32/f64 arithmetic
		return true
	case op >= 0xA8 && op <= 0xAB, op >= 0xAE && op <= 0xBF: // conversions and reinterpretations
		return true
	}
	return false
}

func isDivRemOp(op byte) bool {
	return (op >= 0x6D && op <= 0x70) || (op >= 0x7F && op <= 0x82)
}

// Check parses a wasm module (already uncompressed) and returns nil when it
// passes, or an error wrapping one of the Err* reasons above.
func Check(code []byte) error {
	return check(code, nil)
}

// Inspect is Check that also reports what it saw. For tools and tests.
func Inspect(code []byte) (Stats, error) {
	var st Stats
	err := check(code, &st)
	return st, err
}

func check(code []byte, st *Stats) error {
	if len(code) < 8 || string(code[:4]) != "\x00asm" {
		return fmt.Errorf("%w: missing wasm header", ErrMalformed)
	}
	if binary.LittleEndian.Uint32(code[4:8]) != 1 {
		return fmt.Errorf("%w: unsupported wasm version", ErrMalformed)
	}
	r := &reader{b: code, pos: 8}
	var types []funcType
	sawType := false
	for !r.eof() {
		id, err := r.byte()
		if err != nil {
			return err
		}
		size, err := r.u32()
		if err != nil {
			return fmt.Errorf("%w: section size", ErrMalformed)
		}
		body, err := r.bytes(int(size))
		if err != nil {
			return fmt.Errorf("%w: section %d runs past the end", ErrMalformed, id)
		}
		switch {
		case id == secType:
			if sawType {
				return fmt.Errorf("%w: duplicate type section", ErrMalformed)
			}
			sawType = true
			if types, err = parseTypes(body); err != nil {
				return err
			}
		case id == secCode:
			if err := checkCode(body, types, st); err != nil {
				return err
			}
		case id == secCustom:
			if st != nil {
				st.CustomBytes += len(body)
			}
			// A custom section must still carry a well-formed name.
			sr := &reader{b: body}
			n, err := sr.u32()
			if err != nil {
				return fmt.Errorf("%w: custom section name", ErrMalformed)
			}
			if _, err := sr.bytes(int(n)); err != nil {
				return fmt.Errorf("%w: custom section name", ErrMalformed)
			}
		case id > secDataCount:
			return fmt.Errorf("%w: unknown section id %d", ErrMalformed, id)
		}
	}
	return nil
}

func parseTypes(body []byte) ([]funcType, error) {
	r := &reader{b: body}
	n, err := r.u32()
	if err != nil {
		return nil, fmt.Errorf("%w: type count", ErrMalformed)
	}
	// Each entry is at least 3 bytes; refuse counts the section cannot hold
	// before allocating.
	if uint64(n)*3 > uint64(len(body)) {
		return nil, fmt.Errorf("%w: type count", ErrMalformed)
	}
	types := make([]funcType, 0, n)
	for i := uint32(0); i < n; i++ {
		form, err := r.byte()
		if err != nil {
			return nil, fmt.Errorf("%w: type %d", ErrMalformed, i)
		}
		if form != 0x60 {
			return nil, fmt.Errorf("%w: type %d is not a function type", ErrMalformed, i)
		}
		var ft funcType
		for j := 0; j < 2; j++ {
			cnt, err := r.u32()
			if err != nil {
				return nil, fmt.Errorf("%w: type %d", ErrMalformed, i)
			}
			vals, err := r.bytes(int(cnt))
			if err != nil {
				return nil, fmt.Errorf("%w: type %d", ErrMalformed, i)
			}
			for _, v := range vals {
				if !isNumType(v) {
					return nil, fmt.Errorf("%w: type %d has value type 0x%02x", ErrMalformed, i, v)
				}
			}
			if j == 0 {
				ft.params = cnt
			} else {
				ft.results = cnt
			}
		}
		types = append(types, ft)
	}
	if !r.eof() {
		return nil, fmt.Errorf("%w: trailing bytes in type section", ErrMalformed)
	}
	return types, nil
}

func checkCode(body []byte, types []funcType, st *Stats) error {
	r := &reader{b: body}
	n, err := r.u32()
	if err != nil {
		return fmt.Errorf("%w: function count", ErrMalformed)
	}
	if uint64(n) > uint64(len(body)) {
		return fmt.Errorf("%w: function count", ErrMalformed)
	}
	for i := uint32(0); i < n; i++ {
		size, err := r.u32()
		if err != nil {
			return fmt.Errorf("%w: body %d size", ErrMalformed, i)
		}
		fb, err := r.bytes(int(size))
		if err != nil {
			return fmt.Errorf("%w: body %d runs past the section", ErrMalformed, i)
		}
		if st != nil {
			st.Functions++
		}
		if err := checkBody(fb, types, st); err != nil {
			return fmt.Errorf("function body %d: %w", i, err)
		}
	}
	if !r.eof() {
		return fmt.Errorf("%w: trailing bytes in code section", ErrMalformed)
	}
	return nil
}

type frame struct {
	op      byte
	paramIf bool // an if whose type-index signature has parameters
	sawElse bool
}

func checkBody(body []byte, types []funcType, st *Stats) error {
	r := &reader{b: body}
	groups, err := r.u32()
	if err != nil {
		return fmt.Errorf("%w: locals", ErrMalformed)
	}
	for g := uint32(0); g < groups; g++ {
		if _, err := r.u32(); err != nil {
			return fmt.Errorf("%w: locals", ErrMalformed)
		}
		t, err := r.byte()
		if err != nil || !isNumType(t) {
			return fmt.Errorf("%w: local type", ErrMalformed)
		}
	}

	// The function body itself is the outermost frame.
	stack := []frame{{op: opBlock}}
	var typed error // first type-index block seen; reported once the body is walked
	for {
		if r.eof() {
			return fmt.Errorf("%w: body ends inside a block", ErrMalformed)
		}
		at := r.pos
		op, _ := r.byte()
		if st != nil {
			st.Ops++
			st.ByClass[opClass(op)]++
			switch {
			case IsFloatOp(op):
				st.FloatOps++
			case isDivRemOp(op):
				st.DivRemOps++
			case op == 0x3F || op == 0x40:
				st.MemoryOps++
			}
		}
		switch {
		case op == 0x00 || op == 0x01 || op == 0x0F || op == 0x1A || op == 0x1B:
			// unreachable, nop, return, drop, select
		case op == opBlock || op == opLoop || op == opIf:
			f := frame{op: op}
			idx, isIndex, err := readBlockType(r)
			if err != nil {
				return err
			}
			if isIndex {
				if uint64(idx) >= uint64(len(types)) {
					return fmt.Errorf("%w: block type index %d out of range", ErrMalformed, idx)
				}
				if st != nil {
					st.TypedBlocks++
					if types[idx].params > 0 {
						st.ParamBlocks++
					}
				}
				if typed == nil {
					typed = fmt.Errorf("%w at offset %d", ErrTypedBlock, at)
				}
				f.paramIf = op == opIf && types[idx].params > 0
			}
			stack = append(stack, f)
		case op == opElse:
			top := &stack[len(stack)-1]
			if len(stack) == 1 || top.op != opIf || top.sawElse {
				return fmt.Errorf("%w: else without if at offset %d", ErrMalformed, at)
			}
			top.sawElse = true
		case op == opEnd:
			top := stack[len(stack)-1]
			if top.paramIf && !top.sawElse {
				return fmt.Errorf("%w at offset %d", ErrParamIfWithoutElse, at)
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				if !r.eof() {
					return fmt.Errorf("%w: bytes after the function's end", ErrMalformed)
				}
				return typed
			}
		case op == 0x0C || op == 0x0D: // br, br_if
			if _, err := r.u32(); err != nil {
				return fmt.Errorf("%w: branch depth", ErrMalformed)
			}
		case op == 0x0E: // br_table
			cnt, err := r.u32()
			if err != nil {
				return fmt.Errorf("%w: br_table", ErrMalformed)
			}
			if st != nil && int(cnt) > st.MaxBrTable {
				st.MaxBrTable = int(cnt)
			}
			if cnt > MaxBrTableTargets {
				return fmt.Errorf("%w: %d targets at offset %d, at most %d", ErrBrTableTooLarge, cnt, at, MaxBrTableTargets)
			}
			for k := uint32(0); k <= cnt; k++ { // targets plus the default
				if _, err := r.u32(); err != nil {
					return fmt.Errorf("%w: br_table", ErrMalformed)
				}
			}
		case op == 0x10: // call
			if _, err := r.u32(); err != nil {
				return fmt.Errorf("%w: call", ErrMalformed)
			}
		case op == 0x11: // call_indirect type, table
			if _, err := r.u32(); err != nil {
				return fmt.Errorf("%w: call_indirect", ErrMalformed)
			}
			tbl, err := r.u32()
			if err != nil || tbl != 0 {
				return fmt.Errorf("%w: call_indirect table at offset %d", ErrUnsupportedOpcode, at)
			}
		case op >= 0x20 && op <= 0x24: // local.get/set/tee, global.get/set
			if _, err := r.u32(); err != nil {
				return fmt.Errorf("%w: variable index", ErrMalformed)
			}
		case op >= 0x28 && op <= 0x3E: // loads and stores
			align, err := r.u32()
			if err != nil {
				return fmt.Errorf("%w: memarg", ErrMalformed)
			}
			if align >= 0x40 { // the multi-memory flag
				return fmt.Errorf("%w: memarg with memory index at offset %d", ErrUnsupportedOpcode, at)
			}
			if _, err := r.u32(); err != nil {
				return fmt.Errorf("%w: memarg", ErrMalformed)
			}
		case op == 0x3F || op == 0x40: // memory.size, memory.grow
			mem, err := r.u32()
			if err != nil || mem != 0 {
				return fmt.Errorf("%w: memory index at offset %d", ErrUnsupportedOpcode, at)
			}
		case op == 0x41: // i32.const
			if _, err := r.signed(32, 5); err != nil {
				return fmt.Errorf("%w: i32.const", ErrMalformed)
			}
		case op == 0x42: // i64.const
			if _, err := r.signed(64, 10); err != nil {
				return fmt.Errorf("%w: i64.const", ErrMalformed)
			}
		case op == 0x43: // f32.const
			if _, err := r.bytes(4); err != nil {
				return fmt.Errorf("%w: f32.const", ErrMalformed)
			}
		case op == 0x44: // f64.const
			if _, err := r.bytes(8); err != nil {
				return fmt.Errorf("%w: f64.const", ErrMalformed)
			}
		case op >= 0x45 && op <= 0xC4:
			// MVP numeric operators (0x45..0xBF) and sign extension (0xC0..0xC4).
		case op == 0xFC:
			sub, err := r.u32()
			if err != nil {
				return fmt.Errorf("%w: 0xFC prefix", ErrMalformed)
			}
			if sub > 7 { // only the saturating float-to-int conversions
				return fmt.Errorf("%w: 0xFC %d at offset %d", ErrUnsupportedOpcode, sub, at)
			}
			if st != nil {
				st.FloatOps++
			}
		default:
			return fmt.Errorf("%w: 0x%02x at offset %d", ErrUnsupportedOpcode, op, at)
		}
	}
}

// readBlockType reads a block type: 0x40 (empty), a single value type, or a
// non-negative type index (signed LEB128, 33 bits).
func readBlockType(r *reader) (idx uint32, isIndex bool, err error) {
	if r.eof() {
		return 0, false, fmt.Errorf("%w: block type", ErrMalformed)
	}
	c := r.b[r.pos]
	if c == 0x40 || isNumType(c) {
		r.pos++
		return 0, false, nil
	}
	if c&0x80 == 0 && c&0x40 != 0 {
		// A one-byte negative value that is neither 0x40 nor a numeric type:
		// v128 or a reference type, which the CosmWasm feature set excludes.
		return 0, false, fmt.Errorf("%w: block type 0x%02x", ErrUnsupportedOpcode, c)
	}
	v, err := r.signed(33, 5)
	if err != nil || v < 0 || v > math.MaxUint32 {
		return 0, false, fmt.Errorf("%w: block type", ErrMalformed)
	}
	return uint32(v), true, nil
}
