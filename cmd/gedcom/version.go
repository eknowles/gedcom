package main

import "fmt"

// Version is replaced by CI before building in .travis.yml.
const Version = "1"

func runVersionCommand() {
	fmt.Println(Version)
}
