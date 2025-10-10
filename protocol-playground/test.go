package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
)

func main() {
	b64str, err := generateChallengeKey()
	if err != nil {
		log.Fatal(err)
	}
	randBytes, err := base64.StdEncoding.DecodeString(b64str)
	if err != nil {
		log.Fatal(err)
	}

	strRes := string(randBytes)
	fmt.Println(strRes)
}

func generateChallengeKey() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}
