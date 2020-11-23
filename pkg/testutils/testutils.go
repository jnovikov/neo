package testutils

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func LessString(v1, v2 string) bool {
	return v1 < v2
}

func CanceledContext() context.Context {
	ctx, c := context.WithCancel(context.Background())
	defer c()
	return ctx
}

func TimeoutedContext() context.Context {
	ctx, c := context.WithTimeout(context.Background(), time.Second*0)
	defer c()
	return ctx
}

func RandomStringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func RandomString(length int) string {
	return RandomStringWithCharset(length, charset)
}

func RandomInt(min, max int) int {
	n := seededRand.Intn(max - min)
	return min + n
}

func RandomIP() string {
	gen := func() int {
		return RandomInt(0, 256)
	}
	return fmt.Sprintf("%d.%d.%d.%d", gen(), gen(), gen(), gen())
}
