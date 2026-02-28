package workflows

import "testing"

func TestIsGitMergeConflict(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "conflict text",
			err:  testErr("CONFLICT (content): Merge conflict in file.txt"),
			want: true,
		},
		{
			name: "automatic merge failed",
			err:  testErr("Automatic merge failed; fix conflicts and then commit the result."),
			want: true,
		},
		{
			name: "other error",
			err:  testErr("permission denied"),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isGitMergeConflict(tc.err)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

type testErr string

func (e testErr) Error() string {
	return string(e)
}
