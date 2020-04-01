package pio

// Catch calls the given function returns its error or recovers an Error panic. If an Error panic is
// recovered, its Cause is returned.
func Catch(doer func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if c, ok := r.(*Error); ok {
				err = c.Cause
			} else {
				panic(r)
			}
		}
	}()
	err = doer()
	return
}
