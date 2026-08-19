// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package vc

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
)

type meetingActionDryRunEnvelope struct {
	Data struct {
		API []struct {
			Method string                 `json:"method"`
			URL    string                 `json:"url"`
			Params map[string]interface{} `json:"params"`
			Body   map[string]interface{} `json:"body"`
		} `json:"api"`
	} `json:"data"`
}

func decodeMeetingActionDryRun(t *testing.T, raw []byte) meetingActionDryRunEnvelope {
	t.Helper()

	var envelope meetingActionDryRunEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("unmarshal dry-run output: %v\nraw=%s", err, string(raw))
	}
	if len(envelope.Data.API) != 1 {
		t.Fatalf("dry-run api calls = %d, want 1\nraw=%s", len(envelope.Data.API), string(raw))
	}
	return envelope
}

func registerDryRunNetworkTrap(reg *httpmock.Registry) *httpmock.Stub {
	trap := &httpmock.Stub{Reusable: true, Optional: true}
	reg.Register(trap)
	return trap
}

func assertNoDryRunNetworkCalls(t *testing.T, trap *httpmock.Stub) {
	t.Helper()
	if len(trap.CapturedBodies) != 0 {
		t.Fatalf("dry-run made %d API call(s), want zero", len(trap.CapturedBodies))
	}
}

func TestMeetingAgentActions_DryRunWireContracts(t *testing.T) {
	t.Run("meeting start", func(t *testing.T) {
		f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
		trap := registerDryRunNetworkTrap(reg)

		err := mountAndRun(t, VCMeetingStart, []string{
			"+meeting-start", "--as", "bot",
			"--meeting-number", "123456789",
			"--dry-run", "--format", "json",
		}, f, stdout)
		if err != nil {
			t.Fatalf("mountAndRun() error = %v", err)
		}

		call := decodeMeetingActionDryRun(t, stdout.Bytes()).Data.API[0]
		if call.Method != "POST" || call.URL != meetingBotStartPath {
			t.Fatalf("dry-run call = %s %s, want POST %s", call.Method, call.URL, meetingBotStartPath)
		}
		if len(call.Params) != 0 {
			t.Fatalf("dry-run params = %#v, want none", call.Params)
		}
		if call.Body["join_type"].(float64) != 1 || call.Body["action"].(float64) != 2 {
			t.Fatalf("dry-run body = %#v, want join_type=1 action=2", call.Body)
		}
		identify, _ := call.Body["join_identify"].(map[string]interface{})
		if identify["meeting_no"] != "123456789" {
			t.Fatalf("dry-run join_identify.meeting_no = %v, want 123456789", identify["meeting_no"])
		}
		assertNoDryRunNetworkCalls(t, trap)
	})

	t.Run("meeting invite selected", func(t *testing.T) {
		f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
		trap := registerDryRunNetworkTrap(reg)

		err := mountAndRun(t, VCMeetingInvite, []string{
			"+meeting-invite", "--as", "bot",
			"--meeting-id", "7628568141510692381",
			"--scope", "selected",
			"--invitee-id-type", "union_id",
			"--invitee-ids", "onion_1,onion_2",
			"--dry-run", "--format", "json",
		}, f, stdout)
		if err != nil {
			t.Fatalf("mountAndRun() error = %v", err)
		}

		call := decodeMeetingActionDryRun(t, stdout.Bytes()).Data.API[0]
		if call.Method != "POST" || call.URL != meetingBotInvitePath {
			t.Fatalf("dry-run call = %s %s, want POST %s", call.Method, call.URL, meetingBotInvitePath)
		}
		if call.Params["user_id_type"] != "union_id" {
			t.Fatalf("dry-run params.user_id_type = %v, want union_id", call.Params["user_id_type"])
		}
		if call.Body["meeting_id"] != "7628568141510692381" || call.Body["invite_type"].(float64) != 2 {
			t.Fatalf("dry-run body = %#v, want meeting_id and invite_type=2", call.Body)
		}
		if _, ok := call.Body["scope"]; ok {
			t.Fatalf("dry-run body must not include scope, got %#v", call.Body["scope"])
		}
		invitees, ok := call.Body["invitees"].([]interface{})
		if !ok || len(invitees) != 2 {
			t.Fatalf("dry-run invitees = %#v, want two invitees", call.Body["invitees"])
		}
		first, _ := invitees[0].(map[string]interface{})
		if first["id"] != "onion_1" || first["user_type"].(float64) != 1 {
			t.Fatalf("dry-run first invitee = %#v, want id onion_1 user_type 1", first)
		}
		assertNoDryRunNetworkCalls(t, trap)
	})

	t.Run("meeting invite all", func(t *testing.T) {
		f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
		trap := registerDryRunNetworkTrap(reg)

		err := mountAndRun(t, VCMeetingInvite, []string{
			"+meeting-invite", "--as", "bot",
			"--meeting-id", "7628568141510692381",
			"--scope", "all",
			"--dry-run", "--format", "json",
		}, f, stdout)
		if err != nil {
			t.Fatalf("mountAndRun() error = %v", err)
		}

		call := decodeMeetingActionDryRun(t, stdout.Bytes()).Data.API[0]
		if call.Method != "POST" || call.URL != meetingBotInvitePath {
			t.Fatalf("dry-run call = %s %s, want POST %s", call.Method, call.URL, meetingBotInvitePath)
		}
		if len(call.Params) != 0 {
			t.Fatalf("dry-run params = %#v, want none", call.Params)
		}
		if call.Body["meeting_id"] != "7628568141510692381" || call.Body["invite_type"].(float64) != 1 {
			t.Fatalf("dry-run body = %#v, want meeting_id and invite_type=1", call.Body)
		}
		if _, ok := call.Body["invitees"]; ok {
			t.Fatalf("dry-run body must not include invitees for scope all, got %#v", call.Body["invitees"])
		}
		if _, ok := call.Body["scope"]; ok {
			t.Fatalf("dry-run body must not include scope, got %#v", call.Body["scope"])
		}
		assertNoDryRunNetworkCalls(t, trap)
	})

	t.Run("meeting end", func(t *testing.T) {
		f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
		trap := registerDryRunNetworkTrap(reg)

		err := mountAndRun(t, VCMeetingEnd, []string{
			"+meeting-end", "--as", "bot",
			"--meeting-id", "7628568141510692381",
			"--dry-run", "--format", "json",
		}, f, stdout)
		if err != nil {
			t.Fatalf("mountAndRun() error = %v", err)
		}

		call := decodeMeetingActionDryRun(t, stdout.Bytes()).Data.API[0]
		if call.Method != "POST" || call.URL != meetingBotEndPath {
			t.Fatalf("dry-run call = %s %s, want POST %s", call.Method, call.URL, meetingBotEndPath)
		}
		if len(call.Params) != 0 {
			t.Fatalf("dry-run params = %#v, want none", call.Params)
		}
		if call.Body["meeting_id"] != "7628568141510692381" {
			t.Fatalf("dry-run body.meeting_id = %v, want 7628568141510692381", call.Body["meeting_id"])
		}
		assertNoDryRunNetworkCalls(t, trap)
	})
}

func TestMeetingStart_Execute_BodyActionStart(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
	stub := &httpmock.Stub{
		Method: "POST",
		URL:    meetingBotStartPath,
		Body: map[string]interface{}{
			"code": 0, "msg": "ok",
			"data": map[string]interface{}{
				"meeting": map[string]interface{}{
					"id":         "7628568141510692381",
					"meeting_no": "123456789",
				},
			},
		},
	}
	reg.Register(stub)

	err := mountAndRun(t, VCMeetingStart, []string{
		"+meeting-start", "--as", "bot", "--meeting-number", "123456789", "--format", "json",
	}, f, stdout)
	if err != nil {
		t.Fatalf("mountAndRun() error = %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(stub.CapturedBody, &req); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if req["join_type"].(float64) != 1 {
		t.Fatalf("join_type = %v, want 1", req["join_type"])
	}
	if req["action"].(float64) != 2 {
		t.Fatalf("action = %v, want 2", req["action"])
	}
	identify, _ := req["join_identify"].(map[string]interface{})
	if identify["meeting_no"] != "123456789" {
		t.Fatalf("join_identify.meeting_no = %v, want 123456789", identify["meeting_no"])
	}
}

func TestMeetingInvite_Execute_SelectedScopeWireMapping(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
	stub := &httpmock.Stub{
		Method: "POST",
		URL:    meetingBotInvitePath + "?user_id_type=union_id",
		Body:   map[string]interface{}{"code": 0, "msg": "ok", "data": map[string]interface{}{}},
	}
	reg.Register(stub)

	err := mountAndRun(t, VCMeetingInvite, []string{
		"+meeting-invite", "--as", "bot",
		"--meeting-id", "7628568141510692381",
		"--scope", "selected",
		"--invitee-id-type", "union_id",
		"--invitee-ids", "onion_1,onion_2",
		"--format", "json",
	}, f, stdout)
	if err != nil {
		t.Fatalf("mountAndRun() error = %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(stub.CapturedBody, &req); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if req["meeting_id"] != "7628568141510692381" {
		t.Fatalf("meeting_id = %v, want 7628568141510692381", req["meeting_id"])
	}
	if req["invite_type"].(float64) != 2 {
		t.Fatalf("invite_type = %v, want 2", req["invite_type"])
	}
	if _, ok := req["scope"]; ok {
		t.Fatalf("body must not include scope, got %#v", req["scope"])
	}
	invitees, ok := req["invitees"].([]interface{})
	if !ok || len(invitees) != 2 {
		t.Fatalf("invitees = %#v, want two invitees", req["invitees"])
	}
	first, _ := invitees[0].(map[string]interface{})
	if first["id"] != "onion_1" || first["user_type"].(float64) != 1 {
		t.Fatalf("first invitee = %#v, want id onion_1 user_type 1", first)
	}
}

func TestMeetingInvite_Execute_AllScopeWireMapping(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
	stub := &httpmock.Stub{
		Method: "POST",
		URL:    meetingBotInvitePath,
		Body: map[string]interface{}{
			"code": 0,
			"msg":  "ok",
			"data": map[string]interface{}{
				"invited_count": 198,
				"failed_count":  2,
				"has_more":      true,
			},
		},
	}
	reg.Register(stub)

	err := mountAndRun(t, VCMeetingInvite, []string{
		"+meeting-invite", "--as", "bot",
		"--meeting-id", "7628568141510692381",
		"--scope", "all",
		"--format", "json",
	}, f, stdout)
	if err != nil {
		t.Fatalf("mountAndRun() error = %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(stub.CapturedBody, &req); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if req["invite_type"].(float64) != 1 {
		t.Fatalf("invite_type = %v, want 1", req["invite_type"])
	}
	if _, ok := req["invitees"]; ok {
		t.Fatalf("all scope must not include invitees, got %#v", req["invitees"])
	}
	if _, ok := req["scope"]; ok {
		t.Fatalf("body must not include scope, got %#v", req["scope"])
	}

	var envelope struct {
		Data struct {
			InvitedCount int  `json:"invited_count"`
			FailedCount  int  `json:"failed_count"`
			HasMore      bool `json:"has_more"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if envelope.Data.InvitedCount != 198 || envelope.Data.FailedCount != 2 || !envelope.Data.HasMore {
		t.Fatalf("output data = %+v, want invited_count=198 failed_count=2 has_more=true", envelope.Data)
	}
}

func TestMeetingInvite_Execute_PrettyShowsBatchCountsAndContinuationHint(t *testing.T) {
	tests := []struct {
		name         string
		hasMore      bool
		wantHint     bool
		unwantedText string
	}{
		{
			name:     "more eligible candidates",
			hasMore:  true,
			wantHint: true,
		},
		{
			name:         "batch is complete",
			hasMore:      false,
			unwantedText: "Run again with --scope all",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
			stub := &httpmock.Stub{
				Method: "POST",
				URL:    meetingBotInvitePath,
				Body: map[string]interface{}{
					"code": 0,
					"msg":  "ok",
					"data": map[string]interface{}{
						"invited_count": 197,
						"failed_count":  3,
						"has_more":      tc.hasMore,
					},
				},
			}
			reg.Register(stub)

			err := mountAndRun(t, VCMeetingInvite, []string{
				"+meeting-invite", "--as", "bot",
				"--meeting-id", "7628568141510692381",
				"--scope", "all",
				"--format", "pretty",
			}, f, stdout)
			if err != nil {
				t.Fatalf("mountAndRun() error = %v", err)
			}

			out := stdout.String()
			for _, want := range []string{"Invited this batch: 197", "Failed this batch: 3"} {
				if !strings.Contains(out, want) {
					t.Fatalf("pretty output missing %q:\n%s", want, out)
				}
			}
			hint := "More eligible candidates remain. Run again with --scope all."
			if tc.wantHint && !strings.Contains(out, hint) {
				t.Fatalf("pretty output missing continuation hint %q:\n%s", hint, out)
			}
			if tc.unwantedText != "" && strings.Contains(out, tc.unwantedText) {
				t.Fatalf("pretty output must not imply more candidates when has_more=false:\n%s", out)
			}
			if strings.Contains(out, "invited all") || strings.Contains(out, "全部邀请") {
				t.Fatalf("pretty output must not claim all candidates were invited:\n%s", out)
			}
			if len(stub.CapturedBodies) != 1 {
				t.Fatalf("invite API calls = %d, want exactly one; CLI must not auto-continue", len(stub.CapturedBodies))
			}
		})
	}
}

func TestMeetingInvite_Validation(t *testing.T) {
	tooMany := make([]string, 201)
	for i := range tooMany {
		tooMany[i] = "ou_user_" + strconv.Itoa(i)
	}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "selected missing id type",
			args: []string{"+meeting-invite", "--as", "bot", "--meeting-id", "7628568141510692381", "--scope", "selected", "--invitee-ids", "ou_1"},
			want: "--invitee-id-type",
		},
		{
			name: "selected missing ids",
			args: []string{"+meeting-invite", "--as", "bot", "--meeting-id", "7628568141510692381", "--scope", "selected", "--invitee-id-type", "open_id"},
			want: "--invitee-ids",
		},
		{
			name: "all with ids",
			args: []string{"+meeting-invite", "--as", "bot", "--meeting-id", "7628568141510692381", "--scope", "all", "--invitee-ids", "ou_1"},
			want: "--scope",
		},
		{
			name: "too many selected ids",
			args: []string{"+meeting-invite", "--as", "bot", "--meeting-id", "7628568141510692381", "--scope", "selected", "--invitee-id-type", "open_id", "--invitee-ids", strings.Join(tooMany, ",")},
			want: "--invitee-ids",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, _, _, _ := cmdutil.TestFactory(t, defaultConfig())
			err := mountAndRun(t, VCMeetingInvite, tc.args, f, nil)
			if err == nil {
				t.Fatal("mountAndRun() error = nil, want validation error")
			}
			var valErr *errs.ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("error = %T %v, want *errs.ValidationError", err, err)
			}
			if valErr.Param != tc.want {
				t.Fatalf("Param = %q, want %q", valErr.Param, tc.want)
			}
		})
	}
}

func TestMeetingEnd_Execute_Body(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
	stub := &httpmock.Stub{
		Method: "POST",
		URL:    meetingBotEndPath,
		Body:   map[string]interface{}{"code": 0, "msg": "ok", "data": map[string]interface{}{}},
	}
	reg.Register(stub)

	err := mountAndRun(t, VCMeetingEnd, []string{
		"+meeting-end", "--as", "bot", "--meeting-id", "7628568141510692381", "--format", "json",
	}, f, stdout)
	if err != nil {
		t.Fatalf("mountAndRun() error = %v", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(stub.CapturedBody, &req); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if req["meeting_id"] != "7628568141510692381" {
		t.Fatalf("meeting_id = %v, want 7628568141510692381", req["meeting_id"])
	}

	var envelope struct {
		Data struct {
			MeetingID string `json:"meeting_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if envelope.Data.MeetingID != "7628568141510692381" {
		t.Fatalf("output data.meeting_id = %q, want 7628568141510692381", envelope.Data.MeetingID)
	}
}
