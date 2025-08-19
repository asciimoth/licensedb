# licensedb
Embeddable database of [SPDX licenses](https://github.com/spdx/license-list-data) as a go library

```go
package main

import "github.com/asciimoth/licensedb"

func main() {
	archive := licensedb.Load()
	fmt.Println(archive.List())
	fmt.Println(archive.Get("GPL-3.0-only"))
}
```


