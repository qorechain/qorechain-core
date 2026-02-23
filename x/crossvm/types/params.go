package types

import "fmt"

const (
	DefaultMaxMessageSize  uint64 = 65536 // 64KB
	DefaultMaxQueueSize    uint32 = 1000
	DefaultQueueTimeoutBlocks int64 = 100 // messages expire after 100 blocks
	DefaultEnabled         bool   = true
)

// Params defines the parameters for the crossvm module.
type Params struct {
	MaxMessageSize     uint64 `json:"max_message_size"`
	MaxQueueSize       uint32 `json:"max_queue_size"`
	QueueTimeoutBlocks int64  `json:"queue_timeout_blocks"`
	Enabled            bool   `json:"enabled"`
}

func DefaultParams() Params {
	return Params{
		MaxMessageSize:     DefaultMaxMessageSize,
		MaxQueueSize:       DefaultMaxQueueSize,
		QueueTimeoutBlocks: DefaultQueueTimeoutBlocks,
		Enabled:            DefaultEnabled,
	}
}

func (p Params) Validate() error {
	if p.MaxMessageSize == 0 {
		return fmt.Errorf("max message size must be positive")
	}
	if p.MaxQueueSize == 0 {
		return fmt.Errorf("max queue size must be positive")
	}
	if p.QueueTimeoutBlocks <= 0 {
		return fmt.Errorf("queue timeout blocks must be positive")
	}
	return nil
}
