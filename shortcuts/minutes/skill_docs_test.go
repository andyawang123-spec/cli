// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT
//
// Capability source test: pins the identity (user/bot) claims made in
// skills/lark-minutes against the AuthTypes actually declared on the
// shortcuts, and pins that the write shortcut this PR opened to bot
// (`+apply-permission`) ships with a reference doc — PR #2278's review found
// it had none, which made the main SKILL.md's own "read the reference before
// using a shortcut" contract unsatisfiable.

package minutes

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

// TestMinutesBotShortcutsIdentityDocsMatchAuthTypes pins that every minutes
// shortcut declared bot-capable in code is documented as such in the
// lark-meeting reference that owns the command guidance.
func TestMinutesBotShortcutsIdentityDocsMatchAuthTypes(t *testing.T) {
	skill := readSkillDoc(t, "skills/lark-meeting/SKILL.md")

	for _, cmd := range []struct {
		name      string
		authTypes []string
		reference string
	}{
		{"+search", MinutesSearch.AuthTypes, "lark-minutes-search.md"},
		{"+detail", MinutesDetail.AuthTypes, "lark-minutes-detail.md"},
		{"+download", MinutesDownload.AuthTypes, "lark-minutes-download.md"},
		{"+apply-permission", MinutesApplyPermission.AuthTypes, "lark-minutes-apply-permission.md"},
	} {
		if !hasAuthType(cmd.authTypes, "bot") {
			t.Errorf("%s AuthTypes = %v, want bot included", cmd.name, cmd.authTypes)
			continue
		}
		if !strings.Contains(skill, "references/"+cmd.reference) {
			t.Errorf("skills/lark-meeting/SKILL.md must link %s to %s", cmd.name, cmd.reference)
		}
		reference := readSkillDoc(t, "skills/lark-meeting/references/"+cmd.reference)
		for _, identity := range []string{"--as user", "--as bot"} {
			if !strings.Contains(reference, identity) {
				t.Errorf("%s must document %s support for %s", cmd.reference, identity, cmd.name)
			}
		}
	}
}

// TestMinutesApplyPermissionHasReference pins that `+apply-permission` (this
// PR's new write-capable-by-bot shortcut) has a dedicated reference doc that
// the Shortcuts table links to, satisfying the SKILL.md-wide
// "read the reference before using any shortcut" contract.
func TestMinutesApplyPermissionHasReference(t *testing.T) {
	if !hasAuthType(MinutesApplyPermission.AuthTypes, "bot") {
		t.Fatalf("MinutesApplyPermission.AuthTypes = %v, want bot included (this PR's contract)", MinutesApplyPermission.AuthTypes)
	}

	reference := readSkillDoc(t, "skills/lark-meeting/references/lark-minutes-apply-permission.md")
	for _, must := range []string{"--as bot", "--as user", "身份", "missing scope"} {
		if !strings.Contains(reference, must) {
			t.Errorf("lark-minutes-apply-permission.md must cover %q (user/bot identity + scope-vs-ACL guidance)", must)
		}
	}

	skill := readSkillDoc(t, "skills/lark-meeting/SKILL.md")
	if !strings.Contains(skill, "references/lark-minutes-apply-permission.md") {
		t.Error("skills/lark-meeting/SKILL.md command table must link +apply-permission to its reference doc")
	}
}

func TestLegacyMinutesSkillRoutesToMeetingSkill(t *testing.T) {
	skill := readSkillDoc(t, "skills/lark-minutes/SKILL.md")
	for _, must := range []string{
		"本技能只用于兼容旧名称，不直接处理业务。",
		"../lark-meeting/SKILL.md",
	} {
		if !strings.Contains(skill, must) {
			t.Errorf("skills/lark-minutes/SKILL.md must preserve compatibility routing %q", must)
		}
	}
}
