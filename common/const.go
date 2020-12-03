package common

const (
	TypeNumber8 = "number8"
	TypeNumber16 = "number16"
	TypeNumber32 = "number32"
	TypeNumber64 = "number64"
	TypeFloat64  = "float64"
	TypeString = "string"
	// TypeBytes     = "bytes"
	TypeAddressV4 = "ipv4"
	TypeAddressV6 = "ipv6"
	TypeMac       = "mac"

	TypeTimestamp = "timestamp"

)


var SqlFields = map[string]string{
	TypeNumber8:    "UInt8",
	TypeNumber16:   "UInt16",
	TypeNumber32:   "UInt32",
	TypeNumber64:   "UInt64",
	TypeFloat64:    "Float64",
	TypeMac:       "UInt64",
	TypeAddressV4: "UInt32",
	TypeAddressV6: "FixedString(16)",
	// TypeBytes:     "",
	TypeString: "String",
	TypeTimestamp: "UInt64",
}

type FieldConverter func(value string)(interface{}, error)
//
//var FieldConverters = map[string]FieldConverter{
//	TypeTimestamp:	TimestampFieldConverter,
//
//}