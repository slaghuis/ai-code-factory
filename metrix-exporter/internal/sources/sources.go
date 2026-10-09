package sources

type Source interface {
	Name() string
	Collect() error
}