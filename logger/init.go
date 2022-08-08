// Package logger has implementation of loggers for [mtglib.Logger] interface.
//
// Please see a description of that interface to get some agreements which are
// used by mtglib.
package logger

// StdLikeLogger is an interface which is close to [log.Logger]. This is
// commonly used by many 3pp tools. While mtglib itself does not need it, it is
// always a good idea to support it and have a transient end to end logging.
type StdLikeLogger interface {
	Printf(format string, args ...interface{})
}
