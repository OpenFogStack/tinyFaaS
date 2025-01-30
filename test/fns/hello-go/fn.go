package main

import "fmt"

func fn(data string, headers map[string]string) (string, error) {
	return "Hello, " + data + " with headers: " + fmt.Sprint(headers), nil
}
