package openapi

import (
	"encoding/json"
	"fmt"
	"log"
)

// merge merges the two JSON-marshalable values x1 and x2,
// preferring x1 over x2 except where x1 and x2 are
// JSON objects, in which case the keys from both objects
// are included and their values merged recursively.
//
// It returns an error if x1 or x2 cannot be JSON-marshaled.
func merge(x1, x2 interface{}) (interface{}, error) {
	data1, err := json.Marshal(x1)
	if err != nil {
		return nil, err
	}
	data2, err := json.Marshal(x2)
	if err != nil {
		return nil, err
	}
	var j1 interface{}
	err = json.Unmarshal(data1, &j1)
	if err != nil {
		return nil, err
	}
	var j2 interface{}
	err = json.Unmarshal(data2, &j2)
	if err != nil {
		return nil, err
	}
	return merge1(j1, j2), nil
}

func merge1(x1, x2 interface{}) interface{} {
	switch x1 := x1.(type) {
	case map[string]interface{}:
		x2, ok := x2.(map[string]interface{})
		if !ok {
			return x1
		}
		for k, v2 := range x2 {
			if v1, ok := x1[k]; ok {
				x1[k] = merge1(v1, v2)
			} else {
				x1[k] = v2
			}
		}
	case nil:
		// merge(nil, map[string]interface{...}) -> map[string]interface{...}
		x2, ok := x2.(map[string]interface{})
		if ok {
			return x2
		}
	}
	return x1
}

// Example:

type T struct {
	D string
	C string
	B string
	A string
}

type S struct {
	T T
	W W
}

type SP struct {
	T map[string]int
	W P
}

type W struct {
	X int
}

type P struct {
	X string
	Y int
}

func main() {
	x1 := S{
		T: T{
			D: "d",
			C: "c",
			B: "b",
			A: "a",
		},
		W: W{1},
	}
	x2 := SP{
		T: map[string]int{
			"B": 123,
			"E": 456,
		},
		W: P{
			X: "three",
			Y: 99,
		},
	}

	show(x1, x2)
	fmt.Printf("--------------\n")
	show(x2, x1)
}

func show(x1, x2 interface{}) {
	x3, err := merge(x1, x2)
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(x3, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
}

