// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package vc

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
)

func assertMeetingScreenshotProblem(t *testing.T, err error, wantCategory errs.Category, wantSubtype errs.Subtype) {
	t.Helper()
	if err == nil {
		t.Fatal("expected typed error")
	}
	problem, ok := errs.ProblemOf(err)
	if !ok {
		t.Fatalf("expected typed problem, got %T: %v", err, err)
	}
	if problem.Category != wantCategory || problem.Subtype != wantSubtype {
		t.Fatalf("problem = (%s, %s), want (%s, %s)", problem.Category, problem.Subtype, wantCategory, wantSubtype)
	}
}

func TestVCMeetingScreenshot_Validation(t *testing.T) {
	f, _, _, _ := cmdutil.TestFactory(t, defaultConfig())
	err := mountAndRun(t, VCMeetingScreenshot, []string{
		"+meeting-screenshot", "--as", "user", "--meeting-id", "123456789",
	}, f, nil)
	assertMeetingScreenshotProblem(t, err, errs.CategoryValidation, errs.SubtypeInvalidArgument)
	if !strings.Contains(err.Error(), "long meeting_id") {
		t.Fatalf("error = %v, want long meeting_id validation", err)
	}

	f, _, _, _ = cmdutil.TestFactory(t, defaultConfig())
	err = mountAndRun(t, VCMeetingScreenshot, []string{
		"+meeting-screenshot", "--as", "user", "--meeting-id", "9876543210123", "--timeout", "0s",
	}, f, nil)
	assertMeetingScreenshotProblem(t, err, errs.CategoryValidation, errs.SubtypeInvalidArgument)
	if !strings.Contains(err.Error(), "positive duration") {
		t.Fatalf("error = %v, want positive timeout validation", err)
	}
}

func TestVCMeetingScreenshot_SavesJPEG(t *testing.T) {
	chdirForTest(t)
	image := []byte{0xff, 0xd8, 0x00, 0xff, 0xd9}
	f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
	stub := &httpmock.Stub{
		Method:  http.MethodPost,
		URL:     vcMeetingScreenshotAPIPath,
		RawBody: image,
		Headers: http.Header{
			"Content-Type": {"image/jpeg"},
			"X-Tt-Logid":   {"log-screenshot"},
		},
	}
	reg.Register(stub)

	err := mountAndRun(t, VCMeetingScreenshot, []string{
		"+meeting-screenshot", "--as", "user", "--meeting-id", "9876543210123", "--output", "shot.jpg",
	}, f, stdout)
	if err != nil {
		t.Fatalf("run screenshot command: %v", err)
	}
	reg.Verify(t)
	if got, err := os.ReadFile("shot.jpg"); err != nil || string(got) != string(image) {
		t.Fatalf("saved screenshot = %v, %v; want %v", got, err, image)
	}
	var request map[string]string
	if err := json.Unmarshal(stub.CapturedBody, &request); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if request["meeting_id"] != "9876543210123" {
		t.Fatalf("meeting_id = %q", request["meeting_id"])
	}
	if !strings.Contains(stdout.String(), "log-screenshot") || !strings.Contains(stdout.String(), "shot.jpg") {
		t.Fatalf("output = %s", stdout.String())
	}
	digest := sha256.Sum256(image)
	if !strings.Contains(stdout.String(), fmt.Sprintf("%x", digest)) {
		t.Fatalf("output is missing SHA-256: %s", stdout.String())
	}
}

func TestVCMeetingScreenshot_RejectsInvalidJPEGWithoutReplacingExistingFile(t *testing.T) {
	chdirForTest(t)
	if err := os.WriteFile("shot.jpg", []byte("existing"), 0600); err != nil {
		t.Fatalf("create existing output: %v", err)
	}
	f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
	reg.Register(&httpmock.Stub{
		Method:      http.MethodPost,
		URL:         vcMeetingScreenshotAPIPath,
		RawBody:     []byte{0xff, 0xd8, 0x00},
		ContentType: "image/jpeg",
	})

	err := mountAndRun(t, VCMeetingScreenshot, []string{
		"+meeting-screenshot", "--as", "user", "--meeting-id", "9876543210123", "--output", "shot.jpg", "--overwrite",
	}, f, stdout)
	assertMeetingScreenshotProblem(t, err, errs.CategoryInternal, errs.SubtypeInvalidResponse)
	if !strings.Contains(err.Error(), "invalid screenshot JPEG") {
		t.Fatalf("error = %v, want JPEG validation", err)
	}
	if got, readErr := os.ReadFile("shot.jpg"); readErr != nil || string(got) != "existing" {
		t.Fatalf("existing output changed to %q, err=%v", got, readErr)
	}
}

func TestVCMeetingScreenshot_RequiresOverwriteForExistingFile(t *testing.T) {
	chdirForTest(t)
	if err := os.WriteFile("shot.jpg", []byte("existing"), 0600); err != nil {
		t.Fatalf("create existing output: %v", err)
	}
	f, stdout, _, _ := cmdutil.TestFactory(t, defaultConfig())
	err := mountAndRun(t, VCMeetingScreenshot, []string{
		"+meeting-screenshot", "--as", "user", "--meeting-id", "9876543210123", "--output", "shot.jpg",
	}, f, stdout)
	if err == nil || !strings.Contains(err.Error(), "use --overwrite") {
		t.Fatalf("error = %v, want overwrite requirement", err)
	}
	if got, readErr := os.ReadFile("shot.jpg"); readErr != nil || string(got) != "existing" {
		t.Fatalf("existing output changed to %q, err=%v", got, readErr)
	}
}
