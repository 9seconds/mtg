package mtglib

import "syscall"

// acceptedConnErrnos are the errors Linux accept(2) passes up from the new
// connection itself rather than from the listener. accept(2) says to treat
// them like EAGAIN and retry; Go's poller retries only ECONNABORTED.
var acceptedConnErrnos = []syscall.Errno{
	syscall.ENETDOWN,
	syscall.EPROTO,
	syscall.ENOPROTOOPT,
	syscall.EHOSTDOWN,
	syscall.ENONET,
	syscall.EHOSTUNREACH,
	syscall.EOPNOTSUPP,
	syscall.ENETUNREACH,
}
