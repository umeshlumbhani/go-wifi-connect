package models

// NM80211ApFlags defines access points flag
type NM80211ApFlags uint32

// NM80211ApFlags access point flags
const (
	ApFlagsNone    NM80211ApFlags = 0x0000_0000
	ApFlagsPrivacy NM80211ApFlags = 0x0000_0001
	ApFlagsWPS     NM80211ApFlags = 0x0000_0002
	ApFlagsWPSPBC  NM80211ApFlags = 0x0000_0004
	ApFlagsWPSPIN  NM80211ApFlags = 0x0000_0008
)

// U32 returns value as uint32
func (d NM80211ApFlags) U32() uint32 {
	return uint32(d)
}

// NM80211ApSecurityFlags access point security and authentication flags.
// These flags describe the current security requirements of an access point as determined from the access point's beacon
type NM80211ApSecurityFlags uint32

// NM80211ApSecurityFlags
const (
	ApSecNone                NM80211ApSecurityFlags = 0x0000_0000
	ApSecPairWEP40           NM80211ApSecurityFlags = 0x0000_0001
	ApSecPairWEP104          NM80211ApSecurityFlags = 0x0000_0002
	ApSecPairTKIP            NM80211ApSecurityFlags = 0x0000_0004
	ApSecPairCCMP            NM80211ApSecurityFlags = 0x0000_0008
	ApSecGroupWEP40          NM80211ApSecurityFlags = 0x0000_0010
	ApSecGroupWEP104         NM80211ApSecurityFlags = 0x0000_0020
	ApSecGroupTKIP           NM80211ApSecurityFlags = 0x0000_0040
	ApSecGroupCCMP           NM80211ApSecurityFlags = 0x0000_0080
	ApSecKeyMGMTPSK          NM80211ApSecurityFlags = 0x0000_0100
	ApSecKeyMgmt8021X        NM80211ApSecurityFlags = 0x0000_0200
	ApSecKeyMgmtSAE          NM80211ApSecurityFlags = 0x0000_0400
	ApSecKeyMgmtOWE          NM80211ApSecurityFlags = 0x0000_0800
	ApSecKeyMgmtOWETM        NM80211ApSecurityFlags = 0x0000_1000
	ApSecKeyMgmtEAPSuiteB192 NM80211ApSecurityFlags = 0x0000_2000
)

// U32 used to convert into uint32
func (d NM80211ApSecurityFlags) U32() uint32 {
	return uint32(d)
}
