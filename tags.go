package udprobe

// Tags is a map of attributes and values.
//
// It is defined as a type alias to map[string]string to allow seamless
// integration with other string maps (like prometheus.Labels) without
// needing explicit type conversions.
type Tags = map[string]string

// TagSet is a collection of Tags, indexed by a key (e.g., an IP address).
//
// To avoid panics when setting tags on a new key, use the Set method:
//
//	ts := make(TagSet)
//	ts.Set("1.2.3.4", "dst_hostname", "localhost")
type TagSet map[string]Tags

// Set safely assigns a value to a tag for the given key. It ensures the
// inner Tags map is initialized if it doesn't already exist.
func (ts TagSet) Set(key, tag, value string) {
	if ts[key] == nil {
		ts[key] = make(Tags)
	}
	ts[key][tag] = value
}

// Get returns the Tags for the given key. If no tags exist for the key,
// it returns an empty (but non-nil) Tags map.
func (ts TagSet) Get(key string) Tags {
	if t, ok := ts[key]; ok && t != nil {
		return t
	}
	return make(Tags)
}