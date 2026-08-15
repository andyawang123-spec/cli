// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT
//
// Capability source test: pins the identity (user/bot) claims made in
// skills/lark-vc against the AuthTypes actually declared on the shortcuts.
// PR #2278's review found the docs had already drifted from the code once
// (SKILL.md claimed `+search` supported bot while vc_search.go stayed
// user-only) — this test fails loudly the next time that happens instead of
// relying on a human re-reading both sides on every AuthTypes change.

package vc

import (
	"os"
	"strings"
	"testing"
)

func hasAuthType(authTypes []string, want string) bool {
	for _, a := range authTypes {
		if a == want {
			return true
		}
	}
	return false
}

func readSkillDoc(t *testing.T, relPath string) string {
	t.Helper()
	data, err := os.ReadFile("../../" + relPath)
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(data)
}

// TestVCSearchIdentityDocsMatchAuthTypes pins that `+search` stays user-only
// in both code and docs. If AuthTypes ever gains "bot", this test forces a
// deliberate update to the SKILL.md/reference wording below instead of
// letting the docs silently fall out of sync.
func TestVCSearchIdentityDocsMatchAuthTypes(t *testing.T) {
	skill := readSkillDoc(t, "skills/lark-vc/SKILL.md")
	reference := readSkillDoc(t, "skills/lark-vc/references/lark-vc-search.md")

	if hasAuthType(VCSearch.AuthTypes, "bot") {
		t.Fatalf("VCSearch.AuthTypes = %v now includes bot; update skills/lark-vc/SKILL.md and lark-vc-search.md wording (and this test) to reflect the new support instead of leaving the user-only claims below", VCSearch.AuthTypes)
	}
	if !strings.Contains(skill, "`+search` 仅支持 `--as user`") {
		t.Error("skills/lark-vc/SKILL.md must state that `+search` only supports --as user (matches VCSearch.AuthTypes)")
	}
	if !strings.Contains(reference, "仅支持 `user` 身份") && !strings.Contains(reference, "仅 `--as user`") {
		t.Error("lark-vc-search.md must state that +search only supports user identity (matches VCSearch.AuthTypes)")
	}
}

// TestVCBotShortcutsIdentityDocsMatchAuthTypes pins that the VC shortcuts this
// PR opened to bot (`+detail`, `+recording`) are both declared bot-capable in
// code and documented as such in the main SKILL.md identity line.
func TestVCBotShortcutsIdentityDocsMatchAuthTypes(t *testing.T) {
	skill := readSkillDoc(t, "skills/lark-vc/SKILL.md")

	for _, cmd := range []struct {
		name      string
		authTypes []string
	}{
		{"+detail", VCDetail.AuthTypes},
		{"+recording", VCRecording.AuthTypes},
	} {
		if !hasAuthType(cmd.authTypes, "bot") {
			t.Errorf("%s AuthTypes = %v, want bot included (this PR's contract)", cmd.name, cmd.authTypes)
			continue
		}
		token := "`" + cmd.name + "`"
		if !strings.Contains(skill, token) {
			t.Errorf("skills/lark-vc/SKILL.md identity section must mention %s alongside its bot support", token)
		}
	}
	if !strings.Contains(skill, "也支持 `--as bot`") {
		t.Error("skills/lark-vc/SKILL.md identity section must state which commands also support --as bot")
	}
}

func TestVCAgentActionDocsMatchShortcuts(t *testing.T) {
	skill := readSkillDoc(t, "skills/lark-vc-agent/SKILL.md")
	references := map[string]string{
		"+meeting-start":  "skills/lark-vc-agent/references/lark-vc-agent-meeting-start.md",
		"+meeting-invite": "skills/lark-vc-agent/references/lark-vc-agent-meeting-invite.md",
		"+meeting-end":    "skills/lark-vc-agent/references/lark-vc-agent-meeting-end.md",
	}
	shortcuts := map[string]commonShortcutDocContract{
		"+meeting-start":  {authTypes: VCMeetingStart.AuthTypes, reference: references["+meeting-start"]},
		"+meeting-invite": {authTypes: VCMeetingInvite.AuthTypes, reference: references["+meeting-invite"]},
		"+meeting-end":    {authTypes: VCMeetingEnd.AuthTypes, reference: references["+meeting-end"]},
	}

	for name, contract := range shortcuts {
		if !hasAuthType(contract.authTypes, "bot") || hasAuthType(contract.authTypes, "user") {
			t.Fatalf("%s AuthTypes = %v, want bot-only docs contract", name, contract.authTypes)
		}
		if !strings.Contains(skill, "`"+name+"`") {
			t.Fatalf("skills/lark-vc-agent/SKILL.md must mention %s", name)
		}
		ref := readSkillDoc(t, contract.reference)
		if !strings.Contains(ref, "lark-cli vc "+name+" --as bot") {
			t.Fatalf("%s must document the bot command path", contract.reference)
		}
	}

	inviteRef := readSkillDoc(t, references["+meeting-invite"])
	for _, want := range []string{
		"--scope selected",
		"--invitee-id-type open_id",
		"--invitee-ids",
		"\"invite_type\": 2",
		"user_id_type=open_id",
		"invited_count",
		"failed_count",
		"has_more",
		"单批最多 200",
		"实时重新计算",
		"已经在会中、正在响铃或正在呼叫",
		"无 page token",
		"不会自动循环",
		"Host 或 Co-host",
	} {
		if !strings.Contains(inviteRef, want) {
			t.Fatalf("meeting invite reference missing %q", want)
		}
	}
	for _, legacy := range []string{"--type", "--open-ids"} {
		if strings.Contains(inviteRef, legacy) {
			t.Fatalf("meeting invite reference must not promote legacy %s", legacy)
		}
	}

	for _, want := range []string{"invited_count", "failed_count", "has_more", "不会自动续邀", "Host 或 Co-host"} {
		if !strings.Contains(skill, want) {
			t.Fatalf("skills/lark-vc-agent/SKILL.md meeting invite guidance missing %q", want)
		}
	}
}

type commonShortcutDocContract struct {
	authTypes []string
	reference string
}
