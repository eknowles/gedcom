package main

import (
	"fmt"
	"log"
	"os"
)

func fatalln(args ...interface{}) {
	log.Fatalln(append([]interface{}{"ERROR:"}, args...)...)
}

func fatalf(format string, args ...interface{}) {
	fatalln(fmt.Sprintf(format, args...))
}

func check(err error) {
	if err != nil {
		fatalln(err)
	}
}

func main() {
	if err := newRootCmd(os.Args[0]).Execute(); err != nil {
		fatalln(err)
	}
}
