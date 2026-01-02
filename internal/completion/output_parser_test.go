package completion

import (
	"reflect"
	"testing"

	"github.com/robottwo/bishop/pkg/shellinput"
)

func TestParseExternalCompletionOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []shellinput.CompletionCandidate
	}{
		{
			name:  "empty",
			input: "\n\t  \n",
			want:  []shellinput.CompletionCandidate{},
		},
		{
			name:  "json array of strings",
			input: `["one","two"]`,
			want: []shellinput.CompletionCandidate{
				{Value: "one"},
				{Value: "two"},
			},
		},
		{
			name:  "json array of objects",
			input: `[{"Value":"a","Display":"A","Description":"first"}]`,
			want: []shellinput.CompletionCandidate{
				{Value: "a", Display: "A", Description: "first"},
			},
		},
		{
			name:  "json object",
			input: `{"Value":"b","Display":"B","Description":"second"}`,
			want: []shellinput.CompletionCandidate{
				{Value: "b", Display: "B", Description: "second"},
			},
		},
		{
			name:  "tab delimited",
			input: "foo\tbar",
			want: []shellinput.CompletionCandidate{
				{Value: "foo", Description: "bar"},
			},
		},
		{
			name:  "colon delimited",
			input: "foo:bar",
			want: []shellinput.CompletionCandidate{
				{Value: "foo", Description: "bar"},
			},
		},
		{
			name:  "url value",
			input: "https://example.com",
			want: []shellinput.CompletionCandidate{
				{Value: "https://example.com"},
			},
		},
		{
			name:  "windows path",
			input: `C:\\path\\file`,
			want: []shellinput.CompletionCandidate{
				{Value: `C:\\path\\file`},
			},
		},
		{
			name:  "ipv6 literal",
			input: "2001:db8::1",
			want: []shellinput.CompletionCandidate{
				{Value: "2001:db8::1"},
			},
		},
		{
			name:  "line by line with json array",
			input: "plain\n[{\"Value\":\"c\",\"Display\":\"C\"}]",
			want: []shellinput.CompletionCandidate{
				{Value: "plain"},
				{Value: "c", Display: "C"},
			},
		},
		{
			name:  "mixed values",
			input: "alpha\nvalue:desc\nitem\tinfo",
			want: []shellinput.CompletionCandidate{
				{Value: "alpha"},
				{Value: "value", Description: "desc"},
				{Value: "item", Description: "info"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseExternalCompletionOutput(tt.input)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected result\nwant: %#v\n got: %#v", tt.want, got)
			}
		})
	}
}
