// files defines a set of abstraction for 'files': an openable entities that
// could be read after.
//
// This is not a file on a filesystem of your local machine, it also can
// include "in memory" files or even remote ones, like HTTP endpoints. If you
// make a GET request to HTTP endpoint, then a body is readable and you can
// consider it as an openable file.
package files
