package licensedb

import "fmt"

func ExampleArchive_Get() {
	archive := Load()
	fmt.Println(archive.Get("GPL-3.0-only"))
}
