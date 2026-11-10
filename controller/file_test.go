package controller

import "testing"

func TestBuildUploadTargetPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		basePath string
		dir      string
		filename string
		wantPath string
		wantDir  string
		wantErr  bool
	}{
		{
			name:     "plain file",
			basePath: "/root",
			filename: "app.log",
			wantPath: "/root/app.log",
			wantDir:  "/root",
		},
		{
			name:     "folder upload",
			basePath: "/root/",
			dir:      "logs/2026",
			filename: "app.log",
			wantPath: "/root/logs/2026/app.log",
			wantDir:  "/root/logs/2026",
		},
		{
			name:     "backslash folder upload",
			basePath: "/root",
			dir:      "logs\\2026",
			filename: "app.log",
			wantPath: "/root/logs/2026/app.log",
			wantDir:  "/root/logs/2026",
		},
		{
			name:     "reject path traversal in dir",
			basePath: "/root",
			dir:      "../tmp",
			filename: "app.log",
			wantErr:  true,
		},
		{
			name:     "reject slash in filename",
			basePath: "/root",
			filename: "../app.log",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotPath, gotDir, err := buildUploadTargetPath(tt.basePath, tt.dir, tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
			if gotDir != tt.wantDir {
				t.Fatalf("dir = %q, want %q", gotDir, tt.wantDir)
			}
		})
	}
}

func TestEnsureEditableContentSize(t *testing.T) {
	t.Setenv("WEBSSH_MAX_EDIT_FILE_BYTES", "4")

	if err := ensureEditableContentSize(4); err != nil {
		t.Fatalf("expected size at limit to pass: %v", err)
	}
	if err := ensureEditableContentSize(5); err == nil {
		t.Fatalf("expected oversized content to fail")
	}
}
