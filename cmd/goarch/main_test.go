package main

import (
	"reflect"
	"testing"
)

func TestMergeBuildTagsKeepsGateTag(t *testing.T) {
	args, tags := mergeBuildTags([]string{"./cmd/api"})

	if tags != buildTag {
		t.Fatalf("tags = %q, want %q", tags, buildTag)
	}
	if !reflect.DeepEqual(args, []string{"./cmd/api"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestMergeBuildTagsWithSeparateTagsFlag(t *testing.T) {
	args, tags := mergeBuildTags([]string{"-tags", "libopus", "./cmd/api"})

	if tags != buildTag+",libopus" {
		t.Fatalf("tags = %q", tags)
	}
	if !reflect.DeepEqual(args, []string{"./cmd/api"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestMergeBuildTagsWithEqualsTagsFlag(t *testing.T) {
	args, tags := mergeBuildTags([]string{"-race", "-tags=libopus,sqlite", "./..."})

	if tags != buildTag+",libopus,sqlite" {
		t.Fatalf("tags = %q", tags)
	}
	if !reflect.DeepEqual(args, []string{"-race", "./..."}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestMergeBuildTagsDeduplicatesAndSplitsSpaceSeparatedTags(t *testing.T) {
	args, tags := mergeBuildTags([]string{"-tags", buildTag + " libopus", "-count=1", "./..."})

	if tags != buildTag+",libopus" {
		t.Fatalf("tags = %q", tags)
	}
	if !reflect.DeepEqual(args, []string{"-count=1", "./..."}) {
		t.Fatalf("args = %#v", args)
	}
}
