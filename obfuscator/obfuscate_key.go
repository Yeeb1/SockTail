package main

import (
	"fmt"
	"os"
)

var xorKey = []byte("747sg^8N0$")

func obfuscateAuthKey(key string) {
	data := []byte(key)
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = b ^ xorKey[i%len(xorKey)]
	}
	
	fmt.Println("Replace the obfuscatedAuthKey variable in main.go with:")
	fmt.Print("var obfuscatedAuthKey = []byte{")
	for i, b := range result {
		if i%12 == 0 {
			fmt.Print("\n\t")
		}
		fmt.Printf("0x%02x, ", b)
	}
	fmt.Println("\n}")
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run obfuscate_key.go <your-tailscale-auth-key>")
		fmt.Println("Example: go run obfuscate_key.go tskey-auth-xxxxx-your-real-key")
		os.Exit(1)
	}
	
	authKey := os.Args[1]
	fmt.Printf("Obfuscating key: %s\n", authKey)
	obfuscateAuthKey(authKey)
}
