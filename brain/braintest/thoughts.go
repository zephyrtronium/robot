package braintest

import "iter"

// Collect gathers all thoughts from a think.
func Collect(it iter.Seq[func(id, suf *[]byte) error]) (ids, sufs []string, err error) {
	var id, suf []byte
	for f := range it {
		if err := f(&id, &suf); err != nil {
			return ids, sufs, err
		}
		ids = append(ids, string(id))
		sufs = append(sufs, string(suf))
	}
	return ids, sufs, nil
}
