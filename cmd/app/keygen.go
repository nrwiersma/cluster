package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v2"
)

func runKeyGen(c *cli.Context) error {
	key := make([]byte, 32)
	n, err := rand.Reader.Read(key)
	if err != nil {
		return errors.Wrap(err, "Error reading random data")
	}
	if n != 32 {
		return errors.Wrap(err, "Couldn't read enough entropy")
	}

	fmt.Println(base64.StdEncoding.EncodeToString(key))

	return nil
}
