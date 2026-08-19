# vc +meeting-invite

通过应用机器人在会中邀请成员。这是一次**写操作**。

本 skill 对应 shortcut：`lark-cli vc +meeting-invite`（调用 `POST /open-apis/vc/v1/bots/invite`）。

## 命令

```bash
# 邀请全部建议成员
lark-cli vc +meeting-invite --as bot --meeting-id <meeting_id> --scope all

# 邀请指定成员
lark-cli vc +meeting-invite --as bot \
  --meeting-id <meeting_id> \
  --scope selected \
  --invitee-id-type open_id \
  --invitee-ids ou_xxx,ou_yyy

# 预览 API 调用
lark-cli vc +meeting-invite --as bot --meeting-id <meeting_id> --scope all --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--meeting-id <id>` | 是 | 长数字会议 ID，不是 9 位会议号 |
| `--scope all\|selected` | 是 | `all` 邀请本批建议成员，单批最多 200 人；`selected` 邀请指定成员 |
| `--invitee-id-type open_id\|union_id\|user_id` | selected 必填 | 指定 `--invitee-ids` 的 ID 类型 |
| `--invitee-ids <ids>` | selected 必填 | 指定成员 ID，支持逗号分隔或重复传入，最多 200 个 |

## Wire 合同

`--scope all`：

```json
{
  "meeting_id": "<meeting_id>",
  "invite_type": 1
}
```

不发送 `invitees`，不发送 `scope`。

`--scope selected --invitee-id-type open_id --invitee-ids ou_xxx,ou_yyy`：

```text
POST /open-apis/vc/v1/bots/invite?user_id_type=open_id
```

```json
{
  "meeting_id": "<meeting_id>",
  "invite_type": 2,
  "invitees": [
    {"id": "ou_xxx", "user_type": 1},
    {"id": "ou_yyy", "user_type": 1}
  ]
}
```

不发送 `scope=selected`。仅使用公开 OpenAPI，不 fallback BAM、OGW 或 internal RPC。

## 返回和续邀

JSON 输出原样保留服务端实际返回的聚合字段，不增加外部用户明细。`failed_count` 是基础字段；服务端协议上线 `invited_count` / `has_more` 后，CLI 无需升级即可展示：

```json
{
  "invited_count": 198,
  "failed_count": 2,
  "has_more": true
}
```

- `failed_count` 是本批失败数。仅当响应实际包含 `invited_count` 时，pretty 才展示本批成功数。
- `all` 单批最多 200 人。服务端在每次调用时实时重新计算候选，并过滤已经在会中、正在响铃或正在呼叫的用户。
- 仅当响应实际包含且设置 `has_more=true` 时，pretty 才提示“仍有符合条件候选，可再次调用 `--scope all`”。
- `has_more` 无 page token，CLI 不会自动循环或自动续邀，也不得把本批结果表述为已邀请全部候选。是否再次调用由用户决定；再次调用时服务端会重新计算候选。

## 校验

- `selected` 必须显式传 `--invitee-id-type` 和 `--invitee-ids`。
- `all` 不能传 `--invitee-id-type` 或 `--invitee-ids`。
- `--invitee-ids` 最多 200 个；重复和空值会在本地归一化后去重/忽略。以这份原始输入去空、去重后的数量选择邀请链路：1 人走单点，2～200 人走批量。
- `--meeting-id` 是长数字会议 ID，9 位会议号会被拒绝。

## 权限和前置条件

- 使用应用身份 `--as bot`。
- 会议必须是日程 VC 会议，且应用机器人必须正在该会议中。
- 会议需要开启会中智能体/Agent 能力开关。
- `selected` 去空、去重后只有 1 个 invitee 时走普通单点邀请语义，普通在会参会人也可能可以邀请该用户。
- `all` 和 2～200 人的 `selected` 走批量/建议名单邀请语义，要求应用机器人是当前 Host 或 Co-host；普通参会 Bot 会被下游权限链路拒绝。

## 常见错误与排查

| 错误现象 | 根本原因 | 解决方案 |
|---------|---------|---------|
| `bot is not in the meeting` | 应用机器人当前不在该会议中 | 先用同一个应用机器人发起或加入该日程会议 |
| `switch for allowing agents to join meetings is disabled` | 会议未开启会中智能体/Agent 能力，或会议 Owner 未命中对应发布门禁 | 确认日程会前 AI/Agent 设置与发布门禁后重试 |
| `no permission` | 当前邀请动作不满足会议权限链路，常见于 `all` 或多人 `selected` 时应用机器人不是 Host / Co-host | 若要一键或批量邀请，先确认应用机器人是当前 Host / Co-host；若只邀请 1 人，改用 `--scope selected` 单点邀请 |

## 参考

- [lark-vc-agent-meeting-join](lark-vc-agent-meeting-join.md) — 入会；传 `--action start` 可启动并加入日程会议
- [lark-vc-agent-meeting-end](lark-vc-agent-meeting-end.md) — 结束会议
