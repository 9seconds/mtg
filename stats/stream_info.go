package stats

import statsd "github.com/smira/go-statsd"

type streamInfo map[string]string

func (s streamInfo) T(key string) statsd.Tag {
	return statsd.StringTag(key, s[key])
}

func (s streamInfo) Reset() {
	for k := range s {
		delete(s, k)
	}
}

func getDirection(isRead bool) string {
	if isRead { // for telegram
		return TagDirectionToClient
	}

	return TagDirectionFromClient
}
