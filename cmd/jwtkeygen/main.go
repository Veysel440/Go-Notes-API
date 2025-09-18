package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("kullanım: jwtkeygen <kid>")
		return
	}
	var b [32]byte
	_, _ = rand.Read(b[:])
	fmt.Printf("%s:%s\n", os.Args[1], hex.EncodeToString(b[:]))
}
