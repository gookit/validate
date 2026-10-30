package validate_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gookit/validate/v2"
)

// pngHeader is the PNG signature plus the IHDR chunk header, enough for
// http.DetectContentType to report image/png.
var pngHeader = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}

// buildUpload returns the request and the parsed *multipart.FileHeader of an upload named
// name carrying content.
func buildUpload(t *testing.T, field, name string, content []byte) (*http.Request, *multipart.FileHeader) {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, name)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err = fw.Write(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err = mw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	req := httptest.NewRequest("POST", "/upload", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if err = req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatalf("parse: %v", err)
	}

	return req, req.MultipartForm.File[field][0]
}

// A struct field holding a *multipart.FileHeader is validated by the file rules; before
// this only a *FormData source was accepted and every struct field failed.
func TestFileRule_structSource(t *testing.T) {
	_, png := buildUpload(t, "file", "avatar.png", pngHeader)
	_, txt := buildUpload(t, "file", "notes.txt", []byte("hello"))

	type form struct {
		File *multipart.FileHeader `validate:"image"`
	}

	if v := validate.Struct(&form{File: png}); !v.Validate() {
		t.Errorf("a png should pass the image rule, errors: %v", v.Errors.All())
	}

	v := validate.Struct(&form{File: txt})
	if v.Validate() {
		t.Error("a text file should not pass the image rule")
	} else if msg := v.Errors.One(); msg != "File value must be an image" {
		t.Errorf("unexpected message: %q", msg)
	}
}

// The same value through a map source.
func TestFileRule_mapSource(t *testing.T) {
	_, png := buildUpload(t, "file", "avatar.png", pngHeader)

	v := validate.Map(map[string]any{"file": png})
	v.StringRule("file", "image")
	if !v.Validate() {
		t.Errorf("a png in a map source should pass, errors: %v", v.Errors.All())
	}

	// a value that is not an uploaded file keeps failing
	v = validate.Map(map[string]any{"file": "/some/file"})
	v.StringRule("file", "isFile")
	if v.Validate() {
		t.Error("a plain string should not pass the isFile rule")
	}
}

// A slice field is checked through the ".*" rule.
func TestFileRule_structSliceSource(t *testing.T) {
	_, png := buildUpload(t, "files", "a.png", pngHeader)
	_, txt := buildUpload(t, "files", "b.txt", []byte("hello"))

	v := validate.Map(map[string]any{"files": []*multipart.FileHeader{png, png}})
	v.StringRule("files.*", "image")
	if !v.Validate() {
		t.Errorf("two png files should pass, errors: %v", v.Errors.All())
	}

	v = validate.Map(map[string]any{"files": []*multipart.FileHeader{png, txt}})
	v.StringRule("files.*", "image")
	if v.Validate() {
		t.Error("a slice holding a text file should not pass")
	}
}

// An empty field follows skipEmpty: no file means the rule is skipped.
func TestFileRule_structNilField(t *testing.T) {
	type form struct {
		File *multipart.FileHeader `validate:"image"`
	}
	if v := validate.Struct(&form{}); !v.Validate() {
		t.Errorf("a nil file field should be skipped, errors: %v", v.Errors.All())
	}
}

// The FormData source keeps working as before.
func TestFileRule_formSourceStillWorks(t *testing.T) {
	req, _ := buildUpload(t, "file", "avatar.png", pngHeader)

	v := validate.Request(req)
	v.StringRule("file", "image")
	if !v.Validate() {
		t.Errorf("the request source should still pass, errors: %v", v.Errors.All())
	}
}
