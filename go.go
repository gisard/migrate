package migrate

import (
	"context"
	"reflect"
)

// GoHandler 存储具体 go 反射处理程序
type GoHandler struct {
	index    int
	function reflect.Value
}

type GoFunc func(ctx context.Context) error

func (g *GoHandler) GetIndex() int {
	return g.index
}

func (g *GoHandler) Exec(ctx context.Context) error {
	values := g.function.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(values) == 0 {
		return nil
	}
	if values[0].IsNil() {
		return nil
	}
	return values[0].Interface().(error)
}

func newGoHandler(index int, function reflect.Value) Handler {
	return &GoHandler{
		index:    index,
		function: function,
	}
}
