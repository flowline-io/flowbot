package main

import (
	"github.com/minio/selfupdate"
	"net/http"
)

func doUpdate(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		return err
	}
	return nil
}
