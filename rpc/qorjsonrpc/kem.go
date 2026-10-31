package qorjsonrpc

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/cosmos/cosmos-sdk/client"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

func init() {
	AddService(func(clientCtx client.Context) interface{} {
		return NewKEMAPI(pqctypes.NewQueryClient(clientCtx))
	})
}

// KEMAPI serves qor_getKEMKey.
type KEMAPI struct {
	query func(ctx context.Context, req *pqctypes.QueryKEMKeyRequest) (*pqctypes.QueryKEMKeyResponse, error)
}

// NewKEMAPI builds the service on a pqc query client.
func NewKEMAPI(qc pqctypes.QueryClient) *KEMAPI {
	return &KEMAPI{query: func(ctx context.Context, req *pqctypes.QueryKEMKeyRequest) (*pqctypes.QueryKEMKeyResponse, error) {
		return qc.KEMKey(ctx, req)
	}}
}

// KEMKeyResult is the qor_getKEMKey answer. public_key is 0x-prefixed hex,
// the way Ethereum tooling expects bytes.
type KEMKeyResult struct {
	Address          string        `json:"address"`
	Found            bool          `json:"found"`
	AlgorithmID      uint32        `json:"algorithm_id,omitempty"`
	AlgorithmName    string        `json:"algorithm_name,omitempty"`
	PublicKey        hexutil.Bytes `json:"public_key,omitempty"`
	RegisteredHeight int64         `json:"registered_height,omitempty"`
}

// GetKEMKey returns the ML-KEM public key an address published, if any. The
// address is bech32 (qor1...) or 0x hex. It is served as qor_getKEMKey.
func (api *KEMAPI) GetKEMKey(ctx context.Context, address string) (*KEMKeyResult, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}
	res, err := api.query(ctx, &pqctypes.QueryKEMKeyRequest{Address: address})
	if err != nil {
		return nil, err
	}
	out := &KEMKeyResult{Address: address, Found: res.Found}
	if res.Found && res.Key != nil {
		out.Address = res.Key.Address
		out.AlgorithmID = res.Key.AlgorithmId
		out.AlgorithmName = res.Key.AlgorithmName
		out.PublicKey = res.Key.PublicKey
		out.RegisteredHeight = res.Key.RegisteredHeight
	}
	return out, nil
}
