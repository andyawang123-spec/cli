// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package vc

import (
	"context"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/stretchr/testify/require"
)

func TestVCMeetingAgentActionsDryRun(t *testing.T) {
	setVCDryRunEnv(t)

	tests := []struct {
		name   string
		args   []string
		assert func(t *testing.T, out string)
	}{
		{
			name: "meeting start",
			args: []string{
				"vc", "+meeting-join",
				"--meeting-number", "123456789",
				"--action", "start",
				"--dry-run",
			},
			assert: func(t *testing.T, out string) {
				require.Equal(t, "POST", clie2e.DryRunGet(out, "api.0.method").String(), "stdout:\n%s", out)
				require.Equal(t, "/open-apis/vc/v1/bots/join", clie2e.DryRunGet(out, "api.0.url").String(), "stdout:\n%s", out)
				require.False(t, clie2e.DryRunGet(out, "api.0.params").Exists(), "stdout:\n%s", out)
				require.Equal(t, int64(1), clie2e.DryRunGet(out, "api.0.body.join_type").Int(), "stdout:\n%s", out)
				require.Equal(t, "123456789", clie2e.DryRunGet(out, "api.0.body.join_identify.meeting_no").String(), "stdout:\n%s", out)
				require.Equal(t, int64(2), clie2e.DryRunGet(out, "api.0.body.action").Int(), "stdout:\n%s", out)
			},
		},
		{
			name: "meeting invite",
			args: []string{
				"vc", "+meeting-invite",
				"--meeting-id", "7628568141510692381",
				"--scope", "selected",
				"--invitee-id-type", "union_id",
				"--invitee-ids", "onion_1,onion_2",
				"--dry-run",
			},
			assert: func(t *testing.T, out string) {
				require.Equal(t, "POST", clie2e.DryRunGet(out, "api.0.method").String(), "stdout:\n%s", out)
				require.Equal(t, "/open-apis/vc/v1/bots/invite", clie2e.DryRunGet(out, "api.0.url").String(), "stdout:\n%s", out)
				require.Equal(t, "union_id", clie2e.DryRunGet(out, "api.0.params.user_id_type").String(), "stdout:\n%s", out)
				require.Equal(t, "7628568141510692381", clie2e.DryRunGet(out, "api.0.body.meeting_id").String(), "stdout:\n%s", out)
				require.Equal(t, int64(2), clie2e.DryRunGet(out, "api.0.body.invite_type").Int(), "stdout:\n%s", out)
				require.Equal(t, "onion_1", clie2e.DryRunGet(out, "api.0.body.invitees.0.id").String(), "stdout:\n%s", out)
				require.Equal(t, int64(1), clie2e.DryRunGet(out, "api.0.body.invitees.0.user_type").Int(), "stdout:\n%s", out)
				require.Equal(t, "onion_2", clie2e.DryRunGet(out, "api.0.body.invitees.1.id").String(), "stdout:\n%s", out)
			},
		},
		{
			name: "meeting end",
			args: []string{
				"vc", "+meeting-end",
				"--meeting-id", "7628568141510692381",
				"--dry-run",
			},
			assert: func(t *testing.T, out string) {
				require.Equal(t, "POST", clie2e.DryRunGet(out, "api.0.method").String(), "stdout:\n%s", out)
				require.Equal(t, "/open-apis/vc/v1/bots/end", clie2e.DryRunGet(out, "api.0.url").String(), "stdout:\n%s", out)
				require.False(t, clie2e.DryRunGet(out, "api.0.params").Exists(), "stdout:\n%s", out)
				require.Equal(t, "7628568141510692381", clie2e.DryRunGet(out, "api.0.body.meeting_id").String(), "stdout:\n%s", out)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			result, err := clie2e.RunCmd(ctx, clie2e.Request{
				Args:      tt.args,
				DefaultAs: "bot",
			})
			require.NoError(t, err)
			result.AssertExitCode(t, 0)

			out := result.Stdout
			require.True(t, clie2e.DryRunGet(out, "").Get("api").IsArray(), "stdout:\n%s", out)
			require.Equal(t, int64(1), clie2e.DryRunGet(out, "api.#").Int(), "stdout:\n%s", out)
			tt.assert(t, out)
		})
	}
}
