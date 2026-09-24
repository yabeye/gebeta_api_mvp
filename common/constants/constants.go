package constants

// MaxAddressesPerCustomer caps how many saved addresses a single user can
// have, to prevent unbounded growth and keep address-selection UI usable.
const MaxAddressesPerCustomer = 3

// MaxDeviceTokensPerUser bounds how many FCM device tokens are kept
// per user, evicting the oldest beyond this limit. This is a coarse
// safety net for token accumulation from reinstalls/rotation — proper
// cleanup should also happen reactively when FCM reports a token as
// no longer registered (see internal/platform/firebase).
const MaxDeviceTokensPerUser = 5
