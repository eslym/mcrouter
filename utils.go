package main

import "io"

func Close(c io.Closer) {
	_ = c.Close()
}
