/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package resource

type Reader struct {
}

func NewReader() (*Reader) {
	return &Reader{}
}

func (r *Reader) ReadFrom(path string) ([]byte, error) {
	return Asset(path)
}

