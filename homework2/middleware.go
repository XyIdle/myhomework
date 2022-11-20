package web

type Middleware func(next HandleFunc) HandleFunc

type MiddlewareBuilder struct{}

func NewBuilder() *MiddlewareBuilder {
	return &MiddlewareBuilder{}
}

func (b *MiddlewareBuilder) Build(s string) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			ctx.RespData = []byte(s)
			next(ctx)
		}
	}
}
