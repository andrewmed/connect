package connect

import (
	"testing"
)

func TestMatch(t *testing.T) {
	v := CalculatePath("/a", "/foo/bar", "/a/baz")
	if v != "/foo/bar/baz" {
		t.Fatal(v)
	}
	v = CalculatePath("/a", "foo/bar", "/a/baz")
	if v != "foo/bar/baz" {
		t.Fatal(v)
	}
	v = CalculatePath("/foo/bar/baz", "/foo/bar/baz/bax", "/foo/bar/baz/1")
	if v != "/foo/bar/baz/bax/1" {
		t.Fatal(v)
	}
	v = CalculatePath("/foo/bar/baz", "foo/bar/baz/bax", "/foo/bar/baz/1")
	if v != "foo/bar/baz/bax/1" {
		t.Fatal(v)
	}
	v = CalculatePath("/", "/", "/foo")
	if v != "/foo" {
		t.Fatal(v)
	}
	v = CalculatePath("/", "", "/foo")
	if v != "foo" {
		t.Fatal(v)
	}
	v = CalculatePath("/foo1/bar/baz", "/foo2/bar/baz", "/foo1/bar/baz/bax")
	if v != "/foo2/bar/baz/bax" {
		t.Fatal(v)
	}
	v = CalculatePath("/foo1/bar/baz", "foo2/bar/baz", "/foo1/bar/baz/bax")
	if v != "foo2/bar/baz/bax" {
		t.Fatal(v)
	}
	v = CalculatePath("/Users/andy/code/golang", "golang", "/Users/andy/code/golang/connect/cmd/connect")
	if v != "golang/connect/cmd/connect" {
		t.Fatal(v)
	}
}
