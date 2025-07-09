package metrics

type Metrics interface {
	// BunpTime wrap prometheus histogram for meaturing func time
	BumpTime(key string, tags ...string) (Endable, error)

	// BumpCount warp prometheus counter for key counting, like request count
	BumpCount(key string, val float64, tags ...string) error
}

type Endable interface {
	// End close the timer
	End()
}
