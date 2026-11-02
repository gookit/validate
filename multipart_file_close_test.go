package validate

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/debug"
	"testing"
)

func TestMultipartHelpersCloseFiles(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("upload", "payload.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Repeat([]byte("readable payload\n"), 256)
	if _, err = part.Write(want); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err = req.ParseMultipartForm(1); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := req.MultipartForm.RemoveAll(); err != nil {
			t.Error(err)
		}
	}()
	data := &FormData{Form: req.MultipartForm.Value, Files: req.MultipartForm.File}
	validation := data.Validation()
	countFDs := func(t *testing.T) int {
		t.Helper()
		for _, dir := range []string{"/dev/fd", "/proc/self/fd"} {
			entries, e := os.ReadDir(dir)
			if e == nil {
				return len(entries)
			}
		}
		t.Skip("descriptor enumeration unavailable on this platform")
		return 0
	}
	previousGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previousGC)
	for _, helper := range []string{"FileBytes", "FileMimeType", "IsFormFile"} {
		t.Run(helper, func(t *testing.T) {
			before := countFDs(t)
			for i := 0; i < 32; i++ {
				switch helper {
				case "FileBytes":
					got, e := data.FileBytes("upload")
					if e != nil || !bytes.Equal(got, want) {
						t.Fatalf("bytes %v %v", len(got), e)
					}
				case "FileMimeType":
					if got := data.FileMimeType("upload"); got != "text/plain" {
						t.Fatalf("MIME %q", got)
					}
				case "IsFormFile":
					if !validation.IsFormFile(data, "upload") {
						t.Fatal("file rejected")
					}
				}
			}
			if delta := countFDs(t) - before; delta > 1 {
				t.Errorf("%s left %d opened file descriptors", helper, delta)
			}
		})
	}
	t.Run("missing file", func(t *testing.T) {
		got, e := data.FileBytes("absent")
		if e != nil || got != nil {
			t.Fatalf("missing bytes %q %v", got, e)
		}
		if data.FileMimeType("absent") != "" || validation.IsFormFile(data, "absent") {
			t.Error("missing file accepted")
		}
	})
	t.Run("still readable", func(t *testing.T) {
		f, e := data.GetFile("upload").Open()
		if e != nil {
			t.Fatal(e)
		}
		defer func() {
			if err := f.Close(); err != nil {
				t.Error(err)
			}
		}()
		got, e := io.ReadAll(f)
		if e != nil || !bytes.Equal(got, want) {
			t.Error("original upload changed")
		}
	})
}
