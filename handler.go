package migrate

import "context"

/*
Handler 处理程序实例，拥有执行索引 index，以及自身执行程序，可单独执行
*/

type Handler interface {
	GetIndex() int
	Exec(ctx context.Context) error
}
