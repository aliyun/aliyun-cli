/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package resource

type Reader struct {
}

func NewReader() *Reader {
	return &Reader{}
}

func (r *Reader) ReadFrom(path string) ([]byte, error) {
	return Asset(path)
}
