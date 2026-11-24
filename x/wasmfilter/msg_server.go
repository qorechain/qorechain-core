package wasmfilter

import (
	"context"
	"math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// DecorateMsgServer wraps wasmd's message server so that every message that
// carries bytecode to store (MsgStoreCode, MsgStoreAndInstantiateContract,
// MsgStoreAndMigrateContract) is refused, while the switch is on, when the
// bytecode fails Check. Messages an account sends and messages a governance
// proposal executes go through the same server, so both are covered. Every
// other message is passed through: the engine decorator covers the calls that
// run stored code.
func (f *Filter) DecorateMsgServer(inner wasmtypes.MsgServer) wasmtypes.MsgServer {
	return &msgServer{MsgServer: inner, f: f}
}

type msgServer struct {
	wasmtypes.MsgServer
	f *Filter
}

// checkUpload decides an upload before wasmd stores it, while the filter's
// switch is on. Below the upload-bounds switch it is the bytecode check, the
// float check and the block budget's upload charge, as before.
//
// From the upload-bounds switch (WithUploadBounds), in this order, so that
// no work is done and no budget is taken for an upload that wasmd would
// refuse anyway:
//
//  1. a sender wasmd's upload access does not allow is left to wasmd, which
//     refuses it with its own error: nothing parsed, nothing charged;
//  2. the transaction must have the gas for the whole upload left, from the
//     declared size (gzip's size field, or the length): wasmd's compile
//     price (3 per byte), the code storage price and the inspection price,
//     plus a margin; otherwise it fails with out of gas before anything is
//     unpacked;
//  3. the module is unpacked once, never past its declared size (and
//     wasmd's limit), and inspected in one pass (inspection charged per
//     byte); gzip bytes that are not exactly one member of the declared
//     size (bytes after it included) are refused, as are a module
//     the check refuses, float operators (from their switch) and custom
//     sections over MaxUploadCustomBytes are refused;
//  4. the block's upload pool must have room for the upload's units (the
//     upload is refused, or its transaction aborted, before any further gas
//     is charged); then the code storage price is charged on the
//     uncompressed size, and only then the block wasm budget's upload
//     units, when compiling is about to run.
func (s *msgServer) checkUpload(goCtx context.Context, sender string, code []byte) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if !s.f.on(ctx) {
		return nil
	}
	bounds := s.f.uploadBounds != nil && s.f.uploadBounds(ctx)
	compiled := bounds && s.f.compiledSize != nil && s.f.compiledSize(ctx)
	var declared uint64
	if bounds {
		if s.f.uploadPermitted != nil && !s.f.uploadPermitted(ctx, sender) {
			return nil
		}
		size, ok := DeclaredUploadSize(code)
		if !ok {
			return nil
		}
		declared = size
		perByte := uint64(CompileGasPerByte + UploadStorageGasPerByte + UploadInspectGasPerByte)
		if compiled {
			perByte += CompiledBytesPerWasmByte / CompiledBytesPerGas
		}
		need := saturatingMul(size, perByte) + UploadGasMargin
		if left := ctx.GasMeter().GasRemaining(); left < need {
			return errorsmod.Wrapf(sdkerrors.ErrOutOfGas,
				"storing %d bytes of code needs %d gas, %d left", size, need, left)
		}
		ctx.GasMeter().ConsumeGas(saturatingMul(size, UploadInspectGasPerByte), "wasm code inspection")
	}
	var insp UploadInspection
	var compiledSize uint64
	if bounds {
		// Never more unpacking than was charged, and the declared size is
		// the real one or the upload is refused.
		var uerr error
		if insp, uerr = InspectUploadBounded(code, declared); uerr != nil {
			return errorsmod.Wrapf(ErrCodeRefused, "%v: declared %d bytes", uerr, declared)
		}
	} else {
		var unpacked bool
		if insp, unpacked = InspectUpload(code); !unpacked {
			return nil
		}
	}
	if insp.Err != nil {
		return errorsmod.Wrap(ErrCodeRefused, insp.Err.Error())
	}
	if s.f.refusesFloats(ctx) && insp.Stats.FloatOps > 0 {
		return errorsmod.Wrapf(ErrCodeRefused, "%v: %d float operators", ErrFloatOpcode, insp.Stats.FloatOps)
	}
	if s.f.refusesShape(ctx) {
		if err := CheckShape(insp.Stats); err != nil {
			return errorsmod.Wrap(ErrCodeRefused, err.Error())
		}
	}
	if bounds {
		if insp.Stats.CustomBytes > MaxUploadCustomBytes {
			return errorsmod.Wrapf(ErrCodeRefused, "%v: %d bytes, at most %d", ErrCustomSections, insp.Stats.CustomBytes, MaxUploadCustomBytes)
		}
		if compiled {
			if insp.Stats.Functions > MaxUploadFunctions {
				return errorsmod.Wrapf(ErrCodeRefused, "%v: %d, at most %d", ErrTooManyFunctions, insp.Stats.Functions, MaxUploadFunctions)
			}
			compiledSize = ConservativeCompiledSize(len(insp.Raw), insp.Stats)
			if compiledSize > MaxUploadCompiledSize {
				return errorsmod.Wrapf(ErrCodeRefused, "%v: about %d bytes, at most %d", ErrCompiledTooLarge, compiledSize, MaxUploadCompiledSize)
			}
		}
		// The block's upload pool is checked before the storage and
		// compiled-size gas is charged: an upload that cannot be compiled
		// in this block is refused at the cost of its inspection only.
		if uf, ok := s.f.budget.(UploadFitter); ok {
			if err := uf.UploadFits(ctx, uploadUnits(len(insp.Raw), compiledSize, compiled)); err != nil {
				return err
			}
		}
		ctx.GasMeter().ConsumeGas(saturatingMul(uint64(len(insp.Raw)), UploadStorageGasPerByte), "wasm code storage")
		if compiled {
			ctx.GasMeter().ConsumeGas(compiledGas(compiledSize), "wasm compiled code storage")
		}
	}
	// Compiling the module is validator CPU too: the block wasm budget
	// charges it by size just before the store.
	if uc, ok := s.f.budget.(UploadCharger); ok {
		if err := uc.ChargeUpload(ctx, uploadUnits(len(insp.Raw), compiledSize, compiled)); err != nil {
			return err
		}
	}
	return nil
}

// uploadUnits is what compiling an upload takes from the block's upload
// pool: its bytes, and from wasm-upload-compiled-size the larger of its
// bytes and its conservative compiled size over CompiledBytesPerUploadUnit.
func uploadUnits(size int, compiledSize uint64, compiled bool) int {
	units := size
	if compiled {
		if c := int(compiledSize / CompiledBytesPerUploadUnit); c > units {
			units = c
		}
	}
	return units
}

// compiledGas is the gas an upload pays for its conservative compiled size:
// one per CompiledBytesPerGas bytes, rounded up.
func compiledGas(compiledSize uint64) uint64 {
	return compiledSize/CompiledBytesPerGas + min(compiledSize%CompiledBytesPerGas, 1)
}

func saturatingMul(a, b uint64) uint64 {
	if b != 0 && a > math.MaxUint64/b {
		return math.MaxUint64
	}
	return a * b
}

func (s *msgServer) StoreCode(goCtx context.Context, msg *wasmtypes.MsgStoreCode) (*wasmtypes.MsgStoreCodeResponse, error) {
	if msg != nil {
		if err := s.checkUpload(goCtx, msg.Sender, msg.WASMByteCode); err != nil {
			return nil, err
		}
	}
	return s.MsgServer.StoreCode(goCtx, msg)
}

func (s *msgServer) StoreAndInstantiateContract(goCtx context.Context, msg *wasmtypes.MsgStoreAndInstantiateContract) (*wasmtypes.MsgStoreAndInstantiateContractResponse, error) {
	if msg != nil {
		if err := s.checkUpload(goCtx, msg.Authority, msg.WASMByteCode); err != nil {
			return nil, err
		}
	}
	return s.MsgServer.StoreAndInstantiateContract(goCtx, msg)
}

func (s *msgServer) StoreAndMigrateContract(goCtx context.Context, msg *wasmtypes.MsgStoreAndMigrateContract) (*wasmtypes.MsgStoreAndMigrateContractResponse, error) {
	if msg != nil {
		if err := s.checkUpload(goCtx, msg.Authority, msg.WASMByteCode); err != nil {
			return nil, err
		}
	}
	return s.MsgServer.StoreAndMigrateContract(goCtx, msg)
}
