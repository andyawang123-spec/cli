// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package vc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
	"github.com/larksuite/cli/internal/client"
	"github.com/larksuite/cli/shortcuts/common"
)

const (
	vcMeetingScreenshotAPIPath = "/open-apis/vc/v1/bots/screenshot"
	meetingScreenshotMaxBytes  = 8 << 20
	defaultMeetingScreenshotTO = 15 * time.Second
)

var meetingScreenshotNow = time.Now

// VCMeetingScreenshot captures the current final composite screenshot of an
// ongoing, recorded meeting and writes the returned JPEG locally.
var VCMeetingScreenshot = common.Shortcut{
	Service:     "vc",
	Command:     "+meeting-screenshot",
	Description: "Capture the current final composite screenshot of a meeting",
	Risk:        "read",
	Scopes:      []string{"vc:meeting.realtime:read"},
	AuthTypes:   []string{"user", "bot"},
	Flags: []common.Flag{
		{Name: "meeting-id", Required: true, Desc: "long meeting ID to capture"},
		{Name: "output", Desc: "relative local JPEG output path"},
		{Name: "overwrite", Type: "bool", Desc: "overwrite existing output file"},
		{Name: "timeout", Default: defaultMeetingScreenshotTO.String(), Desc: "request timeout, for example 15s"},
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		if err := validateMeetingEventsMeetingID(runtime.Str("meeting-id")); err != nil {
			return err
		}
		if _, err := meetingScreenshotTimeout(runtime); err != nil {
			return err
		}
		if _, err := runtime.ResolveSavePath(meetingScreenshotOutputPath(runtime)); err != nil {
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "unsafe output path: %s", err).WithParam("--output").WithCause(err)
		}
		return nil
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		return common.NewDryRunAPI().
			POST(vcMeetingScreenshotAPIPath).
			Body(map[string]interface{}{"meeting_id": strings.TrimSpace(runtime.Str("meeting-id"))}).
			Set("output", meetingScreenshotOutputPath(runtime))
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		meetingID := strings.TrimSpace(runtime.Str("meeting-id"))
		outputPath := meetingScreenshotOutputPath(runtime)
		if _, err := runtime.ResolveSavePath(outputPath); err != nil {
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "unsafe output path: %s", err).WithParam("--output").WithCause(err)
		}
		if !runtime.Bool("overwrite") {
			if _, statErr := runtime.FileIO().Stat(outputPath); statErr == nil {
				return errs.NewValidationError(errs.SubtypeFailedPrecondition, "output file already exists: %s (use --overwrite to replace)", outputPath).WithParam("--output")
			}
		}
		timeout, err := meetingScreenshotTimeout(runtime)
		if err != nil {
			return err
		}

		stopSpinner := runtime.StartSpinner("Capturing meeting screenshot")
		defer stopSpinner()
		resp, err := runtime.DoAPIStream(ctx, &larkcore.ApiReq{
			HttpMethod: http.MethodPost,
			ApiPath:    vcMeetingScreenshotAPIPath,
			Body:       map[string]interface{}{"meeting_id": meetingID},
		}, client.WithTimeout(timeout))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
		if contentType != "image/jpeg" {
			return errs.NewInternalError(errs.SubtypeInvalidResponse, "unexpected screenshot content type %q, want image/jpeg", contentType)
		}
		if resp.ContentLength > meetingScreenshotMaxBytes {
			return errs.NewInternalError(errs.SubtypeInvalidResponse, "screenshot exceeds %d bytes", meetingScreenshotMaxBytes)
		}
		image, err := io.ReadAll(io.LimitReader(resp.Body, meetingScreenshotMaxBytes+1))
		if err != nil {
			return errs.NewNetworkError(errs.SubtypeNetworkTransport, "read screenshot response: %v", err).WithCause(err)
		}
		if !validMeetingScreenshotJPEG(image) {
			return errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid screenshot JPEG response")
		}
		result, err := runtime.FileIO().Save(outputPath, fileio.SaveOptions{
			ContentType:   contentType,
			ContentLength: int64(len(image)),
		}, bytes.NewReader(image))
		if err != nil {
			return common.WrapSaveErrorTyped(err)
		}
		savedPath, err := runtime.ResolveSavePath(outputPath)
		if err != nil {
			return errs.NewInternalError(errs.SubtypeUnknown, "resolve saved screenshot path: %v", err).WithCause(err)
		}
		stopSpinner()
		digest := sha256.Sum256(image)
		runtime.Out(map[string]interface{}{
			"meeting_id":   meetingID,
			"path":         savedPath,
			"size_bytes":   result.Size(),
			"content_type": contentType,
			"sha256":       fmt.Sprintf("%x", digest),
			"log_id":       strings.TrimSpace(resp.Header.Get(larkcore.HttpHeaderKeyLogId)),
		}, nil)
		return nil
	},
}

func meetingScreenshotOutputPath(runtime *common.RuntimeContext) string {
	if outputPath := strings.TrimSpace(runtime.Str("output")); outputPath != "" {
		return outputPath
	}
	meetingID := strings.TrimSpace(runtime.Str("meeting-id"))
	stamp := meetingScreenshotNow().UTC().Format("20060102T150405Z")
	return filepath.Join(".lark-vc", "screenshots", fmt.Sprintf("%s-%s.jpg", meetingID, stamp))
}

func meetingScreenshotTimeout(runtime *common.RuntimeContext) (time.Duration, error) {
	raw := strings.TrimSpace(runtime.Str("timeout"))
	if raw == "" {
		return defaultMeetingScreenshotTO, nil
	}
	timeout, err := time.ParseDuration(raw)
	if err != nil || timeout <= 0 {
		return 0, errs.NewValidationError(errs.SubtypeInvalidArgument, "--timeout must be a positive duration, for example 15s").WithParam("--timeout")
	}
	return timeout, nil
}

func validMeetingScreenshotJPEG(image []byte) bool {
	return len(image) >= 4 && len(image) <= meetingScreenshotMaxBytes &&
		image[0] == 0xff && image[1] == 0xd8 && image[len(image)-2] == 0xff && image[len(image)-1] == 0xd9
}
