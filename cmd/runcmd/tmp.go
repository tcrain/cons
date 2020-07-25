package main

import "fmt"

type c interface{}

type a struct {
	b
}

type b struct {
	val int
}

func main() {
	var i c = a{b: b{val: 1}}

	fmt.Println(i, i.(b))
}
