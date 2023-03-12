package interfaces

// Command represents comand
type Command interface {
	StartDnsmasq(dInt string)
	KillDNSMasq()
}
