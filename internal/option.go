package internal

// Option represents a functional option pattern for any type T
type Option[T any] interface {
	apply(*T)
}

// OptionFunc wraps a function to implement the Option interface
type OptionFunc[T any] func(*T)

func (f OptionFunc[T]) apply(target *T) {
	f(target)
}

// ApplyOptions applies all given options to the target
func ApplyOptions[T any](target *T, options ...Option[T]) {
	for _, option := range options {
		option.apply(target)
	}
}
