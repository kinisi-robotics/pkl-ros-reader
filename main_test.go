package main

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// noGitRoot is a gitRoot stub that simulates running outside any git repository.
func noGitRoot() string { return "" }

// noROSPackage is a rosPackage stub that simulates ros2 not being available.
func noROSPackage(_, _ string) string { return "" }

// fakeResolver returns a resolver whose gitRoot points at tmpDir and whose
// rosPackage always returns "".  Use it when the file lives under
// <tmpDir>/ros/<package>/<rel>.
func fakeResolver(tmpDir string) resolver {
	return resolver{
		gitRoot:    func() string { return tmpDir },
		rosPackage: noROSPackage,
	}
}

// mustWriteFile creates all necessary parent directories and writes content to
// path, failing the test immediately on any error.
func mustWriteFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

// mustParseURL wraps url.Parse and fails the test on error.
func mustParseURL(t *testing.T, raw string) url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	return *u
}

// ── splitPath ─────────────────────────────────────────────────────────────────

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty string", "", []string{}},
		{"single component", "foo", []string{"foo"}},
		{"two components", "foo/bar", []string{"foo", "bar"}},
		{"three components", "foo/bar/baz", []string{"foo", "bar", "baz"}},
		{"leading slash", "/foo/bar", []string{"foo", "bar"}},
		{"trailing slash", "foo/bar/", []string{"foo", "bar"}},
		{"multiple consecutive slashes", "foo//bar", []string{"foo", "bar"}},
		{"only slashes", "///", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPath(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitPath(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ── resolver.resolve ──────────────────────────────────────────────────────────

func TestResolver_EmptyPath_ReturnsError(t *testing.T) {
	r := resolver{gitRoot: noGitRoot, rosPackage: noROSPackage}
	uri := mustParseURL(t, "rospkg:///")
	_, err := r.resolve(uri)
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Errorf("expected *os.PathError, got %T: %v", err, err)
	}
}

func TestResolver_GitRoot_ResolvesFile(t *testing.T) {
	tmpDir := t.TempDir()
	wantPath := filepath.Join(tmpDir, "ros", "my_package", "config", "file.pkl")
	mustWriteFile(t, wantPath, []byte("module content"))

	r := fakeResolver(tmpDir)
	uri := mustParseURL(t, "rospkg:///my_package/config/file.pkl")

	got, err := r.resolve(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != wantPath {
		t.Errorf("resolve() = %q, want %q", got, wantPath)
	}
}

func TestResolver_GitRoot_FileAbsent_FallsBackToROSPackage(t *testing.T) {
	tmpDir := t.TempDir()
	// The file does NOT exist under <tmpDir>/ros/..., so the resolver should
	// fall through to the rosPackage finder.
	installedPath := filepath.Join(tmpDir, "installed", "params.yaml")
	mustWriteFile(t, installedPath, []byte("yaml: content"))

	r := resolver{
		gitRoot:    func() string { return tmpDir },
		rosPackage: func(_, _ string) string { return installedPath },
	}

	uri := mustParseURL(t, "rospkg:///my_package/config/params.yaml")
	got, err := r.resolve(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != installedPath {
		t.Errorf("resolve() = %q, want %q", got, installedPath)
	}
}

func TestResolver_NoGitRoot_ResolvesViaROSPackage(t *testing.T) {
	tmpDir := t.TempDir()
	installedPath := filepath.Join(tmpDir, "robot.json")
	mustWriteFile(t, installedPath, []byte(`{"robot":"kr1"}`))

	r := resolver{
		gitRoot:    noGitRoot,
		rosPackage: func(pkg, rel string) string { return installedPath },
	}

	uri := mustParseURL(t, "rospkg:///my_package/robot.json")
	got, err := r.resolve(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != installedPath {
		t.Errorf("resolve() = %q, want %q", got, installedPath)
	}
}

func TestResolver_GitRootTakesPrecedenceOverROSPackage(t *testing.T) {
	tmpDir := t.TempDir()

	// Both a source-tree copy and an installed copy exist.
	sourcePath := filepath.Join(tmpDir, "ros", "my_package", "config", "file.pkl")
	mustWriteFile(t, sourcePath, []byte("source version"))

	installedPath := filepath.Join(tmpDir, "installed", "file.pkl")
	mustWriteFile(t, installedPath, []byte("installed version"))

	r := resolver{
		gitRoot:    func() string { return tmpDir },
		rosPackage: func(_, _ string) string { return installedPath },
	}

	uri := mustParseURL(t, "rospkg:///my_package/config/file.pkl")
	got, err := r.resolve(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Source tree must win.
	if got != sourcePath {
		t.Errorf("resolve() = %q, want source path %q", got, sourcePath)
	}
}

func TestResolver_PackageNameOnly_NoSubPath(t *testing.T) {
	// URI with only a package name and no trailing path component.
	// packageRelPath will be ""; the rosPackage finder receives ("my_package", "").
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	mustWriteFile(t, target, []byte("content"))

	r := resolver{
		gitRoot: noGitRoot,
		rosPackage: func(pkg, rel string) string {
			if pkg == "my_package" && rel == "" {
				return target
			}
			return ""
		},
	}

	uri := mustParseURL(t, "rospkg:///my_package")
	got, err := r.resolve(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != target {
		t.Errorf("resolve() = %q, want %q", got, target)
	}
}

func TestResolver_NotFound_ReturnsError(t *testing.T) {
	r := resolver{gitRoot: noGitRoot, rosPackage: noROSPackage}
	uri := mustParseURL(t, "rospkg:///missing_package/config/file.pkl")
	_, err := r.resolve(uri)
	if err == nil {
		t.Fatal("expected error for unresolvable URI, got nil")
	}
}

func TestResolver_ROSPackageReceivesCorrectArguments(t *testing.T) {
	// Verify that the resolver passes the right packageName and relPath to the
	// rosPackage finder.
	var gotPkg, gotRel string
	r := resolver{
		gitRoot: noGitRoot,
		rosPackage: func(pkg, rel string) string {
			gotPkg, gotRel = pkg, rel
			return "" // still not found; we only care about the arguments
		},
	}

	uri := mustParseURL(t, "rospkg:///kinisi_description/config/system/default.pkl")
	r.resolve(uri) //nolint:errcheck — result not important here

	if gotPkg != "kinisi_description" {
		t.Errorf("rosPackage called with pkg=%q, want %q", gotPkg, "kinisi_description")
	}
	wantRel := filepath.Join("config", "system", "default.pkl")
	if gotRel != wantRel {
		t.Errorf("rosPackage called with rel=%q, want %q", gotRel, wantRel)
	}
}

// ── rospkgModuleReader ────────────────────────────────────────────────────────

func TestModuleReader_Properties(t *testing.T) {
	r := &rospkgModuleReader{res: defaultResolver}

	if got := r.Scheme(); got != "rospkg" {
		t.Errorf("Scheme() = %q, want %q", got, "rospkg")
	}
	if !r.HasHierarchicalUris() {
		t.Error("HasHierarchicalUris() = false, want true")
	}
	if r.IsGlobbable() {
		t.Error("IsGlobbable() = true, want false")
	}
	if !r.IsLocal() {
		t.Error("IsLocal() = false, want true")
	}
}

func TestModuleReader_ListElements_ReturnsEmpty(t *testing.T) {
	r := &rospkgModuleReader{res: defaultResolver}
	uri := mustParseURL(t, "rospkg:///my_package/config/")
	elems, err := r.ListElements(uri)
	if err != nil {
		t.Fatalf("ListElements() error: %v", err)
	}
	if len(elems) != 0 {
		t.Errorf("ListElements() returned %d element(s), want 0", len(elems))
	}
}

func TestModuleReader_Read_ReturnsStringContents(t *testing.T) {
	tmpDir := t.TempDir()
	wantContent := "module MyModule {}\n"
	mustWriteFile(t,
		filepath.Join(tmpDir, "ros", "my_package", "config", "module.pkl"),
		[]byte(wantContent),
	)

	r := &rospkgModuleReader{res: fakeResolver(tmpDir)}
	uri := mustParseURL(t, "rospkg:///my_package/config/module.pkl")

	got, err := r.Read(uri)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if got != wantContent {
		t.Errorf("Read() = %q, want %q", got, wantContent)
	}
}

func TestModuleReader_Read_NotFound_ReturnsError(t *testing.T) {
	r := &rospkgModuleReader{res: resolver{gitRoot: noGitRoot, rosPackage: noROSPackage}}
	uri := mustParseURL(t, "rospkg:///missing/file.pkl")
	_, err := r.Read(uri)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestModuleReader_Read_ReadFileError(t *testing.T) {
	// resolve() passes when Stat succeeds, but ReadFile can still fail — e.g. if
	// the resolved path is a directory rather than a regular file.
	tmpDir := t.TempDir()
	// Create a directory where a .pkl file would be expected.
	dirPath := filepath.Join(tmpDir, "ros", "my_package", "config", "file.pkl")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}

	r := &rospkgModuleReader{res: fakeResolver(tmpDir)}
	uri := mustParseURL(t, "rospkg:///my_package/config/file.pkl")
	_, err := r.Read(uri)
	if err == nil {
		t.Fatal("expected error when resolved path is a directory, got nil")
	}
}

func TestModuleReader_Read_NestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	wantContent := "amends \"../base.pkl\"\n"
	mustWriteFile(t,
		filepath.Join(tmpDir, "ros", "my_package", "config", "system", "override.pkl"),
		[]byte(wantContent),
	)

	r := &rospkgModuleReader{res: fakeResolver(tmpDir)}
	uri := mustParseURL(t, "rospkg:///my_package/config/system/override.pkl")

	got, err := r.Read(uri)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if got != wantContent {
		t.Errorf("Read() = %q, want %q", got, wantContent)
	}
}

// ── rospkgResourceReader ──────────────────────────────────────────────────────

func TestResourceReader_Properties(t *testing.T) {
	r := &rospkgResourceReader{res: defaultResolver}

	if got := r.Scheme(); got != "rospkg" {
		t.Errorf("Scheme() = %q, want %q", got, "rospkg")
	}
	if !r.HasHierarchicalUris() {
		t.Error("HasHierarchicalUris() = false, want true")
	}
	if r.IsGlobbable() {
		t.Error("IsGlobbable() = true, want false")
	}
}

func TestResourceReader_ListElements_ReturnsEmpty(t *testing.T) {
	r := &rospkgResourceReader{res: defaultResolver}
	uri := mustParseURL(t, "rospkg:///my_package/config/")
	elems, err := r.ListElements(uri)
	if err != nil {
		t.Fatalf("ListElements() error: %v", err)
	}
	if len(elems) != 0 {
		t.Errorf("ListElements() returned %d element(s), want 0", len(elems))
	}
}

func TestResourceReader_Read_ReturnsByteContents(t *testing.T) {
	tmpDir := t.TempDir()
	wantContent := []byte("key: value\nother: 42\n")
	mustWriteFile(t,
		filepath.Join(tmpDir, "ros", "my_package", "config", "params.yaml"),
		wantContent,
	)

	r := &rospkgResourceReader{res: fakeResolver(tmpDir)}
	uri := mustParseURL(t, "rospkg:///my_package/config/params.yaml")

	got, err := r.Read(uri)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if !bytes.Equal(got, wantContent) {
		t.Errorf("Read() = %q, want %q", got, wantContent)
	}
}

func TestResourceReader_Read_BinaryContent(t *testing.T) {
	tmpDir := t.TempDir()
	// Simulate a small binary file (e.g. a mesh or calibration blob).
	wantContent := []byte{0x00, 0x01, 0xFF, 0xFE, 0x42}
	mustWriteFile(t,
		filepath.Join(tmpDir, "ros", "my_package", "meshes", "part.bin"),
		wantContent,
	)

	r := &rospkgResourceReader{res: fakeResolver(tmpDir)}
	uri := mustParseURL(t, "rospkg:///my_package/meshes/part.bin")

	got, err := r.Read(uri)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if !bytes.Equal(got, wantContent) {
		t.Errorf("Read() = %v, want %v", got, wantContent)
	}
}

func TestResourceReader_Read_ViaROSPackage(t *testing.T) {
	tmpDir := t.TempDir()
	wantContent := []byte(`{"robot":"kr1"}`)
	installedPath := filepath.Join(tmpDir, "robot.json")
	mustWriteFile(t, installedPath, wantContent)

	r := &rospkgResourceReader{res: resolver{
		gitRoot:    noGitRoot,
		rosPackage: func(_, _ string) string { return installedPath },
	}}

	uri := mustParseURL(t, "rospkg:///my_package/robot.json")
	got, err := r.Read(uri)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if !bytes.Equal(got, wantContent) {
		t.Errorf("Read() = %q, want %q", got, wantContent)
	}
}

func TestResourceReader_Read_NotFound_ReturnsError(t *testing.T) {
	r := &rospkgResourceReader{res: resolver{gitRoot: noGitRoot, rosPackage: noROSPackage}}
	uri := mustParseURL(t, "rospkg:///missing/file.yaml")
	_, err := r.Read(uri)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
