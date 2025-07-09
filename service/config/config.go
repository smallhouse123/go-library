package config

type Config interface {
	// Get key value from either configMap or vault.
	Get(key string) (interface{}, error)
}
