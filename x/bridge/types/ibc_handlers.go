package types

import "fmt"

// ----- ICS-27: Interchain Accounts (ICA) -----

// ICARegistration is the input to RegisterICA — a request to open an
// interchain account on a remote chain controlled by the local owner
// over the given IBC connection.
type ICARegistration struct {
	// Owner is the bech32 address that controls the remote account.
	// Must be a valid QoreChain account address.
	Owner string `json:"owner"`

	// ConnectionID is the IBC connection over which the ICA is opened.
	// Must reference a connection registered with the IBC core module.
	ConnectionID string `json:"connection_id"`

	// Version is the optional ICA version metadata (JSON-encoded).
	// Empty defaults to the upstream-recommended ICS-27 version string.
	Version string `json:"version,omitempty"`
}

// Validate enforces ICA registration invariants at the public-types boundary.
func (r ICARegistration) Validate() error {
	if r.Owner == "" {
		return fmt.Errorf("ica owner must be non-empty")
	}
	if r.ConnectionID == "" {
		return fmt.Errorf("ica connection_id must be non-empty")
	}
	return nil
}

// ICAHandlerHook is the public interface implemented by the proprietary
// keeper's ICS-27 ICA module integration.
type ICAHandlerHook interface {
	// RegisterICA opens an interchain account; returns the remote
	// chain's account address once the channel handshake completes.
	// The implementation is asynchronous on-chain (the channel may
	// not be open yet); callers should poll via QueryICA.
	RegisterICA(reg ICARegistration) error

	// QueryICA returns the remote account address for (owner, connection),
	// or empty string if the account is not yet open.
	QueryICA(owner, connectionID string) (remoteAddr string, ok bool)

	// SubmitICATx forwards an arbitrary tx to the remote ICA. The
	// implementation packages tx-bytes into an ICS-27 packet and
	// dispatches via the ICA controller channel.
	SubmitICATx(owner, connectionID string, txBytes []byte) error
}

// ----- ICS-29: Fee Middleware -----

// FeePayload is the on-the-wire representation of a relayer-fee
// payment for ICS-29. Three fee types are supported:
//   - RecvFee: paid to the relayer that delivers the packet
//   - AckFee:  paid to the relayer that delivers the acknowledgement
//   - TimeoutFee: paid to the relayer that proves a packet timed out
type FeePayload struct {
	RecvFee    string `json:"recv_fee"` // uqor amount as string for big-int safety
	AckFee     string `json:"ack_fee"`
	TimeoutFee string `json:"timeout_fee"`
	Payee      string `json:"payee"` // QoreChain address that receives the fee
}

// Validate ensures all fee fields are populated and addresses look sane.
// Amounts are validated upstream by the keeper using cosmos-sdk math.
func (p FeePayload) Validate() error {
	if p.RecvFee == "" || p.AckFee == "" || p.TimeoutFee == "" {
		return fmt.Errorf("ics-29 fees must all be specified (use \"0\" if zero)")
	}
	if p.Payee == "" {
		return fmt.Errorf("ics-29 payee must be non-empty")
	}
	return nil
}

// FeeMiddlewareHook is the public interface for the ICS-29 wiring.
// Plugs into the v2.6.3 fee-distribution path so relayer rewards are
// settled through the standard 37/30/20/10/3 split.
type FeeMiddlewareHook interface {
	// EscrowPacketFees locks the FeePayload's fees in the fee module
	// account at packet-send time. Refunded on timeout, paid out on
	// successful delivery + ack.
	EscrowPacketFees(packetID string, fees FeePayload) error

	// PayPacketFee distributes the locked fees to the actual relayers
	// once the packet has been delivered and ack'd.
	PayPacketFee(packetID string, recvRelayer, ackRelayer string) error

	// RefundUnusedFees returns escrowed fees on packet timeout.
	RefundUnusedFees(packetID string) error
}

// ----- ICS-721: NFT-IBC -----

// NFTPacketData is the application-level payload for ICS-721 transfers.
// One packet may carry multiple tokens from the same class.
type NFTPacketData struct {
	// ClassID is the IBC-prefixed denom path of the NFT class.
	ClassID string `json:"class_id"`

	// ClassURI is the optional class metadata URI (off-chain JSON).
	ClassURI string `json:"class_uri,omitempty"`

	// TokenIDs are the specific token IDs being transferred.
	TokenIDs []string `json:"token_ids"`

	// TokenURIs are per-token metadata URIs in the same order as TokenIDs.
	TokenURIs []string `json:"token_uris,omitempty"`

	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Memo     string `json:"memo,omitempty"`
}

// Validate enforces NFT packet invariants at the public-types boundary.
func (p NFTPacketData) Validate() error {
	if p.ClassID == "" {
		return fmt.Errorf("nft class_id must be non-empty")
	}
	if len(p.TokenIDs) == 0 {
		return fmt.Errorf("nft token_ids must contain at least one token")
	}
	if len(p.TokenURIs) > 0 && len(p.TokenURIs) != len(p.TokenIDs) {
		return fmt.Errorf("nft token_uris length (%d) != token_ids length (%d)",
			len(p.TokenURIs), len(p.TokenIDs))
	}
	if p.Sender == "" {
		return fmt.Errorf("nft sender must be non-empty")
	}
	if p.Receiver == "" {
		return fmt.Errorf("nft receiver must be non-empty")
	}
	return nil
}

// NFTHandlerHook is the public interface for ICS-721 wiring. The
// proprietary keeper integrates with the upstream cosmos-sdk x/nft
// (or IBC-Go nft-transfer) module.
type NFTHandlerHook interface {
	// SendNFT sends an NFT transfer over the given channel.
	SendNFT(sourcePort, sourceChannel string, data NFTPacketData) error

	// OnRecvNFTPacket processes an incoming NFT packet.
	OnRecvNFTPacket(data NFTPacketData) error
}
