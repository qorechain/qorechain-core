package app

import (
	"bytes"
	"context"
	"fmt"
	"time"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/textual"
	"github.com/fxamacker/cbor/v2"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// The Textual query service renders SIGN_MODE_TEXTUAL sign documents for
// wallets without an ADR-050 renderer of their own (CosmJS, Keplr). It is
// node-local and needs no protocol switch:
//
//   - It is a query. It runs on the query router over a branch of committed
//     state, reads bank denom metadata and writes nothing; whatever it would
//     write is discarded with the branch.
//   - Nothing in block execution can reach it. CosmWasm contracts get no
//     gRPC or Stargate queries (wasmd's reject queriers), the 08-wasm light
//     clients only reach the explicit wasmClientQueryAllowList, and interchain
//     accounts only reach queries annotated module_query_safe, which this one
//     is not. So it changes no state, gas, event or verdict of any transaction.
//   - It verifies nothing. The chain recomputes the sign bytes itself in the
//     ante handler; a node that answered wrongly would only make the device
//     show the user what it then signs, and the signature would fail.
//
// Work is bounded before and during rendering: the transaction's size and
// message count, the store reads (a gas limit), the number of denoms looked
// up, a deadline checked on each lookup, the screens returned, and how many
// renders run at once on this node.
const (
	TextualQueryMaxTxBytes    = 64 * 1024
	TextualQueryMaxMessages   = 32
	TextualQueryMaxDenoms     = 32
	TextualQueryMaxScreens    = 1024
	TextualQueryGasLimit      = 3_000_000
	TextualQueryTimeout       = 2 * time.Second
	TextualQueryMaxConcurrent = 8

	// Ledger Cosmos app limits (cosmos/ledger-cosmos v2.39.3).
	ledgerMaxTitle   = 40
	ledgerMaxContent = 550
	ledgerMaxScreens = 127
	ledgerMaxDocSize = 16384
)

type textualQueryServer struct {
	txConfig client.TxConfig
	registry codectypes.InterfaceRegistry
	metadata textual.CoinMetadataQueryFn
	slots    chan struct{}
}

func newTextualQueryServer(app *QoreChainApp) *textualQueryServer {
	return &textualQueryServer{
		txConfig: app.txConfig,
		registry: app.interfaceRegistry,
		metadata: authtxconfig.NewBankKeeperCoinMetadataQueryFn(app.BankKeeper),
		slots:    make(chan struct{}, TextualQueryMaxConcurrent),
	}
}

var _ pqctypes.TextualServer = (*textualQueryServer)(nil)

func (s *textualQueryServer) TextualSignDoc(goCtx context.Context, req *pqctypes.QueryTextualSignDocRequest) (resp *pqctypes.QueryTextualSignDocResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		return nil, status.Error(codes.ResourceExhausted, "too many textual renders in progress on this node; retry")
	}

	raw, err := s.rawTx(req)
	if err != nil {
		return nil, err
	}
	auth := &txtypes.AuthInfo{}
	if err := auth.Unmarshal(raw.AuthInfoBytes); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "auth info: %v", err)
	}
	body := &txtypes.TxBody{}
	if err := body.Unmarshal(raw.BodyBytes); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "body: %v", err)
	}
	if err := textualTxShape(body, auth); err != nil {
		return nil, err
	}
	// Decode the transaction as the chain does, with one empty signature per
	// signer (signatures are not part of the sign document).
	unsigned, err := (&txtypes.TxRaw{BodyBytes: raw.BodyBytes, AuthInfoBytes: raw.AuthInfoBytes,
		Signatures: make([][]byte, len(auth.SignerInfos))}).Marshal()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	tx, err := s.txConfig.TxDecoder()(unsigned)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "decode tx: %v", err)
	}
	if n := len(tx.GetMsgs()); n == 0 || n > TextualQueryMaxMessages {
		return nil, status.Errorf(codes.InvalidArgument, "a transaction of %d messages; 1 to %d are rendered", n, TextualQueryMaxMessages)
	}

	if req.PublicKey == nil {
		return nil, status.Error(codes.InvalidArgument, "public_key is required")
	}
	var pub cryptotypes.PubKey
	if err := s.registry.UnpackAny(req.PublicKey, &pub); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "public_key: %v", err)
	}
	addr := sdk.AccAddress(pub.Address()).String()
	if req.SignerAddress != "" && req.SignerAddress != addr {
		return nil, status.Errorf(codes.InvalidArgument, "signer_address %s is not the address of public_key (%s)", req.SignerAddress, addr)
	}
	if err := textualSignerInfo(auth, req.PublicKey, req.Sequence); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx).WithGasMeter(storetypes.NewGasMeter(TextualQueryGasLimit))
	deadline := time.Now().Add(TextualQueryTimeout)
	lookups := 0
	metadata := func(ctx context.Context, denom string) (*bankv1beta1.Metadata, error) {
		lookups++
		if lookups > TextualQueryMaxDenoms {
			return nil, status.Errorf(codes.InvalidArgument, "more than %d denom lookups", TextualQueryMaxDenoms)
		}
		if time.Now().After(deadline) {
			return nil, status.Error(codes.DeadlineExceeded, "textual rendering took too long")
		}
		return s.metadata(ctx, denom)
	}
	h, err := textual.NewSignModeHandler(textual.SignModeOptions{CoinMetadataQuerier: metadata, FileResolver: s.registry})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	defer func() {
		if r := recover(); r != nil {
			if oog, ok := r.(storetypes.ErrorOutOfGas); ok {
				resp, err = nil, status.Errorf(codes.ResourceExhausted, "textual rendering ran out of gas (%s)", oog.Descriptor)
				return
			}
			panic(r)
		}
	}()
	signBytes, err := authsigning.GetSignBytesAdapter(sdkCtx, txsigning.NewHandlerMap(h), signing.SignMode_SIGN_MODE_TEXTUAL,
		authsigning.SignerData{Address: addr, ChainID: sdkCtx.ChainID(), AccountNumber: req.AccountNumber, Sequence: req.Sequence, PubKey: pub}, tx)
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.InvalidArgument, "render: %v", err)
	}
	if time.Now().After(deadline) {
		return nil, status.Error(codes.DeadlineExceeded, "textual rendering took too long")
	}

	screens, err := decodeTextualScreens(signBytes)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if len(screens) > TextualQueryMaxScreens {
		return nil, status.Errorf(codes.InvalidArgument, "%d screens; at most %d are returned", len(screens), TextualQueryMaxScreens)
	}
	issue := ledgerIssue(screens, len(signBytes))
	return &pqctypes.QueryTextualSignDocResponse{
		SignBytes:        signBytes,
		Screens:          screens,
		ChainId:          sdkCtx.ChainID(),
		LedgerCompatible: issue == "",
		LedgerIssue:      issue,
		BodyBytes:        raw.BodyBytes,
		AuthInfoBytes:    raw.AuthInfoBytes,
	}, nil
}

// rawTx reads the request's transaction into its body and auth info bytes.
func (s *textualQueryServer) rawTx(req *pqctypes.QueryTextualSignDocRequest) (*txtypes.TxRaw, error) {
	switch {
	case len(req.TxBytes) > 0 && req.TxJson != "":
		return nil, status.Error(codes.InvalidArgument, "set one of tx_bytes and tx_json")
	case len(req.TxBytes) > TextualQueryMaxTxBytes || len(req.TxJson) > 2*TextualQueryMaxTxBytes:
		return nil, status.Errorf(codes.InvalidArgument, "the transaction exceeds %d bytes", TextualQueryMaxTxBytes)
	case len(req.TxBytes) > 0:
		// A Tx and a TxRaw have the same wire form; reading a TxRaw keeps
		// the body and auth info bytes exactly as the wallet will send them.
		raw := &txtypes.TxRaw{}
		if err := raw.Unmarshal(req.TxBytes); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "tx_bytes: %v", err)
		}
		return raw, nil
	case req.TxJson != "":
		// Check the shape first: the encoder dereferences the body, the auth
		// info and the fee, and an empty or partial transaction ("{}") used
		// to crash the handler instead of being refused.
		var shape txtypes.Tx
		if err := codec.NewProtoCodec(s.registry).UnmarshalJSON([]byte(req.TxJson), &shape); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "tx_json: %v", err)
		}
		if err := textualTxShape(shape.Body, shape.AuthInfo); err != nil {
			return nil, err
		}
		tx, err := s.txConfig.TxJSONDecoder()([]byte(req.TxJson))
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "tx_json: %v", err)
		}
		bz, err := s.txConfig.TxEncoder()(tx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "tx_json: %v", err)
		}
		if len(bz) > TextualQueryMaxTxBytes {
			return nil, status.Errorf(codes.InvalidArgument, "the transaction exceeds %d bytes", TextualQueryMaxTxBytes)
		}
		raw := &txtypes.TxRaw{}
		if err := raw.Unmarshal(bz); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		return raw, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "tx_bytes or tx_json is required")
	}
}

// textualTxShape refuses a transaction the renderer cannot take: one without
// a body, messages, auth info, fee or signer infos. The SDK encoder and
// renderer dereference these, so a partial transaction crashed the handler
// (HTTP 500) instead of being refused. A signer info without a public key
// stays allowed (an account whose key is on chain), and the requester's own
// signer info is checked by textualSignerInfo.
func textualTxShape(body *txtypes.TxBody, auth *txtypes.AuthInfo) error {
	switch {
	case body == nil:
		return status.Error(codes.InvalidArgument, "the transaction has no body")
	case len(body.Messages) == 0:
		return status.Error(codes.InvalidArgument, "the transaction has no messages")
	case auth == nil:
		return status.Error(codes.InvalidArgument, "the transaction has no auth_info")
	case auth.Fee == nil:
		return status.Error(codes.InvalidArgument, "the auth_info has no fee")
	case len(auth.SignerInfos) == 0:
		return status.Error(codes.InvalidArgument, "the auth_info has no signer_infos")
	}
	return nil
}

// textualSignerInfo requires the auth info to name this key as a
// SIGN_MODE_TEXTUAL signer at this sequence. The PQC signature, if any,
// already covers the auth info, so a wallet that got either wrong would sign
// a document the chain refuses.
func textualSignerInfo(auth *txtypes.AuthInfo, pub *codectypes.Any, seq uint64) error {
	for _, si := range auth.SignerInfos {
		if si.PublicKey == nil || si.PublicKey.TypeUrl != pub.TypeUrl || !bytes.Equal(si.PublicKey.Value, pub.Value) {
			continue
		}
		if si.ModeInfo.GetSingle().GetMode() != signing.SignMode_SIGN_MODE_TEXTUAL {
			return status.Error(codes.InvalidArgument, "the auth info signs this key in another mode; set its mode_info to SIGN_MODE_TEXTUAL")
		}
		if si.Sequence != seq {
			return status.Errorf(codes.InvalidArgument, "the auth info has sequence %d for this key, the request %d", si.Sequence, seq)
		}
		return nil
	}
	return status.Error(codes.InvalidArgument, "the auth info has no signer with this public key")
}

func decodeTextualScreens(bz []byte) ([]*pqctypes.TextualScreen, error) {
	var doc map[uint64][]map[uint64]interface{}
	if err := cbor.Unmarshal(bz, &doc); err != nil {
		return nil, fmt.Errorf("decode textual document: %w", err)
	}
	out := make([]*pqctypes.TextualScreen, 0, len(doc[1]))
	for _, m := range doc[1] {
		sc := &pqctypes.TextualScreen{}
		if v, ok := m[1].(string); ok {
			sc.Title = v
		}
		if v, ok := m[2].(string); ok {
			sc.Content = v
		}
		if v, ok := m[3].(uint64); ok {
			sc.Indent = uint32(v)
		}
		if v, ok := m[4].(bool); ok {
			sc.Expert = v
		}
		out = append(out, sc)
	}
	return out, nil
}

func ledgerIssue(screens []*pqctypes.TextualScreen, docLen int) string {
	if len(screens) > ledgerMaxScreens {
		return fmt.Sprintf("%d screens, the Ledger app reviews at most %d", len(screens), ledgerMaxScreens)
	}
	if docLen > ledgerMaxDocSize {
		return fmt.Sprintf("a %d-byte document, the Ledger app takes at most %d", docLen, ledgerMaxDocSize)
	}
	for i, sc := range screens {
		switch {
		case len(sc.Title) > ledgerMaxTitle:
			return fmt.Sprintf("screen %d has a %d-byte title, the Ledger app shows at most %d", i, len(sc.Title), ledgerMaxTitle)
		case len(sc.Content) > ledgerMaxContent:
			return fmt.Sprintf("screen %d has %d bytes of content, the Ledger app shows at most %d", i, len(sc.Content), ledgerMaxContent)
		case sc.Content == "":
			return fmt.Sprintf("screen %d has no content, which the Ledger app refuses", i)
		}
	}
	return ""
}

// registerTextualGateway mounts POST /qorechain/pqc/v1/textual_sign_doc.
func registerTextualGateway(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := pqctypes.RegisterTextualHandlerClient(context.Background(), mux, pqctypes.NewTextualClient(clientCtx)); err != nil {
		panic(err)
	}
}
