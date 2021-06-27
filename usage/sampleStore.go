package main

import (
	"io"

	mime "github.com/Promignis/rfc2822"
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

func newSampleStore() mime.Store {
	return &SampleStore{}
}
