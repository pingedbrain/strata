package main

import "fmt"

// @spec hello/greeting
func greet(name string) string {
	return "hello " + name
}

func main() {
	fmt.Println(greet("strata"))
}
