package main

import (
	"io"
)

type SampleStore struct{}

func (s *SampleStore) GetType() string {
	return ""
}

func (s *SampleStore) Put(key string, reader io.Reader) error {
	return nil
}

func (s *SampleStore) Get(key string) (io.ReadCloser, error) {

	return nil, nil
}
