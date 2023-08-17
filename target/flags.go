package target

type Flags uint64

const (
	FlagPre Flags = 1 << iota
	FlagAbs
	FlagCors
	FlagSecureMode
	FlagForwardHost
	FlagForwardAddr
	FlagIgnoreCert
	FlagWebsocket
)

var (
	routeFlagMask    = FlagPre | FlagAbs | FlagCors | FlagSecureMode | FlagForwardHost | FlagForwardAddr | FlagIgnoreCert | FlagWebsocket
	redirectFlagMask = FlagPre | FlagAbs
)

// HasFlag returns true if the bits contain the requested flag
func (f Flags) HasFlag(flag Flags) bool {
	// 0110 & 0100 == 0100  (value != 0 thus true)
	// 0011 & 0100 == 0000  (value == 0 thus false)
	return f&flag != 0
}

// NormaliseRouteFlags returns only the bits used for routes
func (f Flags) NormaliseRouteFlags() Flags {
	// removes bits outside the mask
	// 0110 & 0111 == 0110
	// 1010 & 0111 == 0010  (values are different)
	return f & routeFlagMask
}

// NormaliseRedirectFlags returns only the bits used for redirects
func (f Flags) NormaliseRedirectFlags() Flags {
	// removes bits outside the mask
	// 0110 & 0111 == 0110
	// 1010 & 0111 == 0010  (values are different)
	return f & redirectFlagMask
}
