package promnats

import (
	"reflect"
	"regexp"
	"testing"
)

func Test_WithParts(t *testing.T) {
	type args struct {
		parts []string
	}
	tests := []struct {
		name    string
		args    args
		want    *options
		wantErr bool
	}{
		{"first", args{[]string{"A", "B", "C"}}, &options{Subjects: []string{"", "a", "a.b", "a.b.c"}}, false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := WithParts(tt.args.parts...)
			got := &options{}
			err := fn(got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Option() = %v, want %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithSubj() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_genId(t *testing.T) {
	type args struct {
		s []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"first", args{[]string{"", "A", "A.B", "A.B.C"}}, "a.b.c"},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := genID(tt.args.s); got != tt.want {
				t.Errorf("genId() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultSubject(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		match   bool
	}{
		{"first", `[a-z\d_-]+\.[a-z\d-_].[\d]+$`, true},
		{"second", `[a-z\d_-]+\.[a-z\d-_].[a-z]+$`, false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultSubject()

			match, err := regexp.MatchString(tt.pattern, got)
			if err != nil {
				t.Fatalf("MatchString error: %v", err)
			}
			if match != tt.match {
				t.Errorf("DefaultSubject() = %v, want match %v, pattern %v", got, tt.match, tt.pattern)
			}
		})
	}
}
