// Command engram은 지식관리 위키의 승급 파이프라인을 다루는 CLI다.
package main

import (
	"os"

	"github.com/neocode24/engram/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
