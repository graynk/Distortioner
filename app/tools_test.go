package main

import (
	"errors"
	"testing"
)

func TestExtractPossibleTimeout(t *testing.T) {
	err := errors.New("telegram: retry after 5 (429)")
	timeout, err := extractPossibleTimeout(err)
	if timeout != 5 || err != nil {
		t.Fatal(err)
	}
	err = errors.New("telegram: retry after 10 (429)")
	timeout, err = extractPossibleTimeout(err)
	if timeout != 10 || err != nil {
		t.Fatal(err)
	}

	err = errors.New("retry after 24 (429)")
	timeout, err = extractPossibleTimeout(err)
	if timeout != 24 || err != nil {
		t.Fatal(err)
	}

	err = errors.New("retry after 50")
	timeout, err = extractPossibleTimeout(err)
	if timeout != 50 || err != nil {
		t.Fatal(err)
	}
	err = errors.New("retry after")
	timeout, err = extractPossibleTimeout(err)
	if timeout != 0 || err == nil {
		t.Fatal(err)
	}
	err = errors.New("telegram: bot was kicked from the supergroup chat (403)")
	timeout, err = extractPossibleTimeout(err)
	if timeout != 0 || err == nil {
		t.Fatal(err)
	}
}
