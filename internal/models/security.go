package models

// SECURITY flag for access point security key management
type SECURITY uint32

// Flags for security
const (
	NONE       SECURITY = 0b0000_0000
	WEP        SECURITY = 0b0000_0001
	WPA        SECURITY = 0b0000_0010
	WPA2       SECURITY = 0b0000_0100
	ENTERPRISE SECURITY = 0b0000_1000
)

// U32 used to return uint32 flag
func (d SECURITY) U32() uint32 {
	return uint32(d)
}

// String returns string
func (d SECURITY) String() string {
	switch d {
	case WEP:
		return "web"
	case WPA:
		return "wpa"
	case WPA2:
		return "wp2"
	case ENTERPRISE:
		return "enterprise"
	default:
		return "none"
	}
}
