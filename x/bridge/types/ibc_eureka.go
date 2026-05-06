package types

import "fmt"

// EurekaPacket models a generic IBC Eureka v2 packet at the type layer.
// The full wire format depends on the upstream cosmos-sdk Eureka v2
// module; this struct captures the fields the bridge module needs to
// route attestations and credit deposits without coupling the public
// types package to a specific upstream version.
//
// Production: when the upstream Eureka v2 module path lands in go.mod,
// the proprietary keeper translates this struct to/from the upstream
// packet representation.
type EurekaPacket struct {
	// SourceChain is the chain ID of the originating chain (e.g.
	// "cosmoshub", "osmosis"). Matches a ChainConfig.ChainID.
	SourceChain string `json:"source_chain"`

	// DestChain is the chain ID of the receiving chain. For deposits
	// into QoreChain, this equals the local chain ID.
	DestChain string `json:"dest_chain"`

	// Sequence is the per-channel monotonically-increasing packet number.
	Sequence uint64 `json:"sequence"`

	// PortID and ChannelID identify the IBC port/channel pair used
	// (Eureka v2 retains compatibility with the classic identifiers).
	PortID    string `json:"port_id"`
	ChannelID string `json:"channel_id"`

	// ClientType names the light client behind the channel. Common
	// values: "tendermint" (catch-all for chains using the upstream
	// SDK consensus engine), "solomachine".
	ClientType string `json:"client_type"`

	// Data is the application-specific payload. For ICS-20 transfers
	// this is a JSON-encoded FungibleTokenPacketData; the bridge
	// keeper unmarshals it on receipt.
	Data []byte `json:"data"`

	// TimeoutTimestamp is the unix-nano timestamp after which the
	// packet is considered timed out. Zero means no timeout.
	TimeoutTimestamp uint64 `json:"timeout_timestamp,omitempty"`
}

// Validate enforces the structural invariants of an Eureka v2 packet
// at the public-types boundary. Called by the proprietary keeper before
// any state-mutating operation.
func (p EurekaPacket) Validate() error {
	if p.SourceChain == "" {
		return fmt.Errorf("source_chain must be non-empty")
	}
	if p.DestChain == "" {
		return fmt.Errorf("dest_chain must be non-empty")
	}
	if p.SourceChain == p.DestChain {
		return fmt.Errorf("source_chain and dest_chain must differ")
	}
	if p.PortID == "" {
		return fmt.Errorf("port_id must be non-empty")
	}
	if p.ChannelID == "" {
		return fmt.Errorf("channel_id must be non-empty")
	}
	if p.ClientType == "" {
		return fmt.Errorf("client_type must be non-empty")
	}
	if p.Sequence == 0 {
		return fmt.Errorf("sequence must be > 0")
	}
	if len(p.Data) == 0 {
		return fmt.Errorf("data must be non-empty")
	}
	return nil
}

// EurekaAck is the application-level acknowledgement returned by the
// receiving chain. Success encodes a positive ack; failure encodes
// the human-readable error.
type EurekaAck struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// IsSuccess returns true if the ack signals successful delivery.
func (a EurekaAck) IsSuccess() bool { return a.Success && a.Error == "" }

// EurekaHandlerHook is the public interface the proprietary keeper's
// Eureka v2 handler implements. The bridge module's gRPC + light-client
// glue calls into this interface so the public-side test surface can
// exercise the dispatch path without depending on the upstream module.
type EurekaHandlerHook interface {
	// OnRecvPacket processes an incoming Eureka v2 packet. Returns the
	// acknowledgement to send back to the source chain.
	OnRecvPacket(packet EurekaPacket) EurekaAck

	// OnAcknowledgement handles the response to a packet we previously
	// sent. The ack may be Success or carry an error string.
	OnAcknowledgement(packet EurekaPacket, ack EurekaAck) error

	// OnTimeout handles a packet that timed out before reaching the
	// destination. Implementation must un-escrow any tokens locked
	// for the original send.
	OnTimeout(packet EurekaPacket) error
}
