package grpcweb

type options struct {
	moveTrailerToHeader bool
}

// Option is an object that controls the behavior of the downgrading gRPC server.
type Option interface {
	apply(o *options)
}

type optionFunc func(o *options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func MoveTrailerToHeader(move bool) Option {
	return optionFunc(func(o *options) {
		o.moveTrailerToHeader = move
	})
}
