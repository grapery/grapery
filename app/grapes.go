package main

import "fmt"
import "flag"
import "github.com/grapery/grapery/version"

var printVersion = flag.Bool("version", false, "app build version")

func main() {
	flag.Parse()
	fmt.Println("app init")
	if *printVersion {
		version.PrintFullVersionInfo()
		return
	}
}
