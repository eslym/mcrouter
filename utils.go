package main

import "io"

func ReadLine(reader io.Reader) <-chan string {
	ch := make(chan string)
	go func() {
		buffer := make([]byte, 1024)
		str := ""
		for {
			n, err := reader.Read(buffer)
			if err != nil {
				if str != "" {
					ch <- str
				}
				close(ch)
				break
			}
			for i := 0; i < n; i++ {
				if buffer[i] == '\n' {
					ch <- str
					str = ""
				} else {
					str += string(buffer[i])
				}
			}
		}
	}()
	return ch
}

func Close(c io.Closer) {
	_ = c.Close()
}
