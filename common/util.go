package common

import (
	"fmt"
	"math/rand"
)

func PrintVersion() {
	const version = "1.0.0"
	fmt.Println("avege version", version)
}

func GenerateRandomString(length int) (res string) {
	baseStr := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	for i := 0; i < length; i++ {
		index := rand.Intn(len(baseStr))
		res = res + string(baseStr[index])
	}
	return
}
