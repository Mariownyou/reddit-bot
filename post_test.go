package main

import (
	"testing"
)

func TestParsePostMessage(t *testing.T) {
	text := "Hello @world @test @test2"
	expectedText := "Hello"
	expectedSubs := []string{"world", "test", "test2"}

	text, subs := ParsePostMessage(text)

	if text != expectedText {
		t.Errorf("Expected text to be %s, got %s", expectedText, text)
	}

	if len(subs) != len(expectedSubs) {
		t.Errorf("Expected subs to be %v, got %v", expectedSubs, subs)
	}

	for i, sub := range subs {
		if sub != expectedSubs[i] {
			t.Errorf("Expected subs to be %v, got %v", expectedSubs, subs)
		}
	}

	text = "Hello world, hahaha😝\n\n@world @test @test2"
	expectedText = "Hello world, hahaha😝"
	expectedSubs = []string{"world", "test", "test2"}

	text, subs = ParsePostMessage(text)
	if text != expectedText {
		t.Errorf("Expected text to be %s, got %s", expectedText, text)
	}

	if len(subs) != len(expectedSubs) {
		t.Errorf("Expected subs to be %v, got %v", expectedSubs, subs)
	}

	for i, sub := range subs {
		if sub != expectedSubs[i] {
			t.Errorf("Expected subs to be %v, got %v", expectedSubs, subs)
		}
	}
}

func TestParseRepostMessage(t *testing.T) {

}
