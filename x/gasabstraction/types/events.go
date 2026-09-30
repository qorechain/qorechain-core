package types

// Events emitted by fee conversion.
const (
	EventTypeFeeConverted = "gas_abstraction_conversion"
	EventTypeFloatRefill  = "gas_abstraction_refill"

	AttributeKeyPaid   = "paid"
	AttributeKeyNative = "native_equivalent"
	AttributeKeyRate   = "rate"
	AttributeKeySource = "price_source"
	AttributeKeyResult = "result"
)
