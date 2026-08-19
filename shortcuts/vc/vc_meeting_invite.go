// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package vc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/shortcuts/common"
)

const (
	meetingBotInvitePath = "/open-apis/vc/v1/bots/invite"

	meetingInviteScopeAll      = "all"
	meetingInviteScopeSelected = "selected"
	meetingInviteeLimit        = 200
	meetingInviteTypeAll       = 1
	meetingInviteTypeSelected  = 2
)

var validInviteeIDTypes = map[string]struct{}{
	"open_id":  {},
	"union_id": {},
	"user_id":  {},
}

// VCMeetingInvite invites selected users or all eligible users as the app bot.
var VCMeetingInvite = common.Shortcut{
	Service:     "vc",
	Command:     "+meeting-invite",
	Description: "Invite selected or all eligible users as the app bot",
	Risk:        "write",
	Scopes:      []string{"vc:meeting.bot.join:write"},
	AuthTypes:   []string{"bot"},
	HasFormat:   true,
	Flags: []common.Flag{
		{Name: "meeting-id", Required: true, Desc: "meeting ID"},
		{Name: "scope", Required: true, Desc: "invite scope", Enum: []string{meetingInviteScopeAll, meetingInviteScopeSelected}},
		{Name: "invitee-id-type", Desc: "selected invitee ID type", Enum: []string{"open_id", "union_id", "user_id"}},
		{Name: "invitee-ids", Type: "string_slice", Desc: "selected invitee IDs, comma-separated or repeated; maximum 200"},
	},
	Normalize: func(_ context.Context, flags *common.FlagContext) error {
		return flags.SetCanonical("scope", strings.ToLower(strings.TrimSpace(flags.Str("scope"))))
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		_, _, err := buildMeetingInviteRequest(runtime)
		return err
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		params, body, err := buildMeetingInviteRequest(runtime)
		if err != nil {
			return common.NewDryRunAPI().Set("error", err.Error())
		}
		return common.NewDryRunAPI().POST(meetingBotInvitePath).Params(params).Body(body)
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		params, body, err := buildMeetingInviteRequest(runtime)
		if err != nil {
			return err
		}
		data, err := runtime.CallAPITyped(http.MethodPost, meetingBotInvitePath, params, body)
		if err != nil {
			return err
		}
		if data == nil {
			data = map[string]interface{}{}
		}
		runtime.OutFormat(data, nil, func(w io.Writer) {
			printMeetingInviteSummary(w, data)
		})
		return nil
	},
}

func buildMeetingInviteRequest(runtime *common.RuntimeContext) (map[string]interface{}, map[string]interface{}, error) {
	if err := validateMeetingEventsMeetingID(runtime.Str("meeting-id")); err != nil {
		return nil, nil, err
	}
	scope := strings.ToLower(strings.TrimSpace(runtime.Str("scope")))
	switch scope {
	case meetingInviteScopeAll:
		if len(normalizeInviteeIDs(runtime.StrSlice("invitee-ids"))) != 0 || strings.TrimSpace(runtime.Str("invitee-id-type")) != "" {
			return nil, nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--invitee-id-type and --invitee-ids must not be set when --scope all").
				WithParam("--scope")
		}
		return nil, map[string]interface{}{
			"meeting_id":  strings.TrimSpace(runtime.Str("meeting-id")),
			"invite_type": meetingInviteTypeAll,
		}, nil
	case meetingInviteScopeSelected:
		idType := strings.TrimSpace(runtime.Str("invitee-id-type"))
		if _, ok := validInviteeIDTypes[idType]; !ok {
			return nil, nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--invitee-id-type must be open_id, union_id, or user_id when --scope selected").
				WithParam("--invitee-id-type")
		}
		ids := normalizeInviteeIDs(runtime.StrSlice("invitee-ids"))
		if len(ids) == 0 {
			return nil, nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--invitee-ids is required when --scope selected").
				WithParam("--invitee-ids")
		}
		if len(ids) > meetingInviteeLimit {
			return nil, nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--invitee-ids accepts at most %d users, got %d", meetingInviteeLimit, len(ids)).
				WithParam("--invitee-ids")
		}
		return map[string]interface{}{"user_id_type": idType}, map[string]interface{}{
			"meeting_id":  strings.TrimSpace(runtime.Str("meeting-id")),
			"invite_type": meetingInviteTypeSelected,
			"invitees":    buildInvitees(ids),
		}, nil
	case "":
		return nil, nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--scope is required").WithParam("--scope")
	default:
		return nil, nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--scope must be all or selected, got %q", runtime.Str("scope")).WithParam("--scope")
	}
}

func normalizeInviteeIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			id := strings.TrimSpace(part)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func buildInvitees(ids []string) []map[string]interface{} {
	invitees := make([]map[string]interface{}, 0, len(ids))
	for _, id := range ids {
		invitees = append(invitees, map[string]interface{}{
			"id":        id,
			"user_type": 1,
		})
	}
	return invitees
}

func printMeetingInviteSummary(w io.Writer, data map[string]interface{}) {
	fmt.Fprintln(w, "Invite request sent.")
	if count, ok := common.GetFloatOK(data, "invited_count"); ok {
		fmt.Fprintf(w, "  Invited this batch: %d\n", int(count))
	}
	if count, ok := common.GetFloatOK(data, "failed_count"); ok {
		fmt.Fprintf(w, "  Failed this batch: %d\n", int(count))
	}
	if common.GetBool(data, "has_more") {
		fmt.Fprintln(w, "More eligible candidates remain. Run again with --scope all.")
	}
}
