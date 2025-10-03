package main

import (
	"encoding/hex"
	"fmt"
	"os"
)

var xorKey = []byte("747sg^8N0$")

func obfuscateAuthKeyToHex(key string) string {
	data := []byte(key)
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = b ^ xorKey[i%len(xorKey)]
	}
	return hex.EncodeToString(result)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <auth-key>\n", os.Args[0])
		os.Exit(1)
	}

	authKey := os.Args[1]
	fmt.Print(obfuscateAuthKeyToHex(authKey))
}