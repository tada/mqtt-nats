package pio

// Catch calls the given function returns its error or recovers an Error panic. If an Error panic is
// recovered, its Cause is returned.
func Catch(doer func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case *Error:
				err = r.Cause
			case Error:
				err = r.Cause
			default:
				panic(r)
			}
		}
	}()
	err = doer()
	return
}
