package tooling

import (
	"testing"
)

func TestApplyPatch(t *testing.T) {
	tests := []struct {
		name     string
		original string
		patch    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Simple addition",
			original: "line1\nline2\n",
			patch: `--- a/file
+++ b/file
@@ -1,2 +1,3 @@
 line1
 line2
+line3`,
			expected: "line1\nline2\nline3\n",
		},
		{
			name:     "Simple removal",
			original: "line1\nline2\nline3\n",
			patch: `--- a/file
+++ b/file
@@ -1,3 +1,2 @@
 line1
-line2
 line3`,
			expected: "line1\nline3\n",
		},
		{
			name:     "Multiple hunks",
			original: "A\nB\nC\nD\nE\nF\nG\n",
			patch: `--- a/file
+++ b/file
@@ -1,3 +1,3 @@
 A
-B
+X
 C
@@ -5,3 +5,3 @@
 E
-F
+Y
 G`,
			expected: "A\nX\nC\nD\nE\nY\nG\n",
		},
		{
			name:     "Context mismatch",
			original: "line1\nwrong\nline3\n",
			patch: `--- a/file
+++ b/file
@@ -1,3 +1,3 @@
 line1
-line2
+new
 line3`,
			wantErr: true,
		},
		{
			name:     "Hunk starting further down",
			original: "1\n2\n3\n4\n5\n6\n",
			patch: `@@ -4,3 +4,4 @@
 4
 5
 6
+7`,
			expected: "1\n2\n3\n4\n5\n6\n7\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyPatch(tt.original, tt.patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("applyPatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("applyPatch() got = %q, want %q", got, tt.expected)
			}
		})
	}
}
