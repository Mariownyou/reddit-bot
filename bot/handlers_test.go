package bot

import (
	"testing"
	"reflect"
)

var m = Manager{
	Data: NewContext(),
}

func TestParsePost(t *testing.T) {
	m.ParsePost("Title: Hello world\nhello: test, message\nempty: , success")
	if m.Data.flairs["hello"] != "test" {
		t.Error("flair not found for sub hello", m.Data.flairs["hello"])
	}

	if m.Data.flairs["empty"] != "" {
		t.Error("flair not found for sub empty", m.Data.flairs["empty"])
	}
}

func TestParseFailedPost(t *testing.T) {
	failed := m.ParseFailedPost("Title: Hello world\nhello: test, ❌message\nempty: , ❌success\nsome: test, message")
	if len(failed) != 2 {
		t.Error("failed to parse failed post")
	}

	if !reflect.DeepEqual(failed[0], []string{"hello", "test"}) {
		t.Error("expected hello to be failed, got", failed[0])
	}

	if !reflect.DeepEqual(failed[1], []string{"empty", ""}) {
		t.Error("expected empty to be failed, got", failed[1])
	}
}

func TestParseFailedPost2(t *testing.T) {
	post := `Title: asdasd
	test: , ❌ Debug mode, post not submitted`

	failed := m.ParseFailedPost(post)
	t.Log("failed:", failed)
}
