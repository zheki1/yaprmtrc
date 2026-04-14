package buildinfo

import "fmt"

var (
	Version string
	Date    string
	Commit  string
)

func Print() {
	v := Version
	if v == "" {
		v = "N/A"
	}
	d := Date
	if d == "" {
		d = "N/A"
	}
	c := Commit
	if c == "" {
		c = "N/A"
	}
	fmt.Printf("Build version: %s\n", v)
	fmt.Printf("Build date: %s\n", d)
	fmt.Printf("Build commit: %s\n", c)
}
