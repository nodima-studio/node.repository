package main

import (
	"encoding/base64"
	"strings"
	"testing"

	runnerv1 "github.com/nodima-studio/nodima-sdk/runner/v1"
)

func TestTransformEncodesBytesAndPreservesNulls(t *testing.T) {
	t.Parallel()
	batch, err := transform(runnerv1.Batch{
		RowCount: 2,
		Columns: []runnerv1.Column{{
			Name: "payload", Type: runnerv1.DataTypeBytes,
			Bytes: [][]byte{[]byte("hello world"), []byte("ignored")},
			Valid: []bool{true, false},
		}},
	}, settings{column: "payload", action: actionEncode, encoding: base64.URLEncoding})
	if err != nil {
		t.Fatal(err)
	}
	column := batch.Columns[0]
	if column.Type != runnerv1.DataTypeString {
		t.Fatalf("type = %q, want string", column.Type)
	}
	if got, want := column.String, []string{"aGVsbG8gd29ybGQ=", ""}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("values = %q, want %q", got, want)
	}
	if got, want := column.Valid, []bool{true, false}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("validity = %v, want %v", got, want)
	}
}

func TestTransformDecodesAndReportsInvalidBase64(t *testing.T) {
	t.Parallel()
	configured := settings{column: "payload", action: actionDecode, encoding: base64.StdEncoding}
	batch, err := transform(runnerv1.Batch{Columns: []runnerv1.Column{{
		Name: "payload", Type: runnerv1.DataTypeString, String: []string{"aGVsbG8="},
	}}}, configured)
	if err != nil {
		t.Fatal(err)
	}
	if got := batch.Columns[0].String[0]; got != "hello" {
		t.Fatalf("decoded value = %q, want hello", got)
	}
	_, err = transform(runnerv1.Batch{Columns: []runnerv1.Column{{
		Name: "payload", Type: runnerv1.DataTypeString, String: []string{"not-base64!!!"},
	}}}, configured)
	if err == nil || !strings.Contains(err.Error(), "row 0") {
		t.Fatalf("error = %v, want invalid Base64 row error", err)
	}
}
