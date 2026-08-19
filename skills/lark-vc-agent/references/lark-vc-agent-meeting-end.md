# vc +meeting-end

通过应用机器人结束一场会议。这是一次**写操作**，会影响所有参会人。

本 skill 对应 shortcut：`lark-cli vc +meeting-end`（调用 `POST /open-apis/vc/v1/bots/end`）。

## 命令

```bash
lark-cli vc +meeting-end --as bot --meeting-id <meeting_id>
lark-cli vc +meeting-end --as bot --meeting-id <meeting_id> --format json
lark-cli vc +meeting-end --as bot --meeting-id <meeting_id> --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--meeting-id <id>` | 是 | 长数字会议 ID，不是 9 位会议号 |
| `--dry-run` | 否 | 预览 API 调用，不实际结束会议 |

## Wire 合同

```json
{
  "meeting_id": "<meeting_id>"
}
```

仅使用公开 OpenAPI `POST /open-apis/vc/v1/bots/end`。不 fallback BAM、OGW 或 internal RPC。

## 返回

`--format json` 的 `data.meeting_id` 会回显本次传入的长数字会议 ID，便于后续日志和自动化步骤关联已结束的会议。

## 核心约束

- 使用应用身份 `--as bot`。
- `--meeting-id` 必须是长数字会议 ID；9 位会议号会被拒绝。
- 这是影响整场会议的写操作，只在用户明确要求结束会议时调用。

## 常见错误与排查

| 错误现象 | 根本原因 | 解决方案 |
|---------|---------|---------|
| `bot is not in the meeting` | 应用机器人当前不在该会议中 | 先用同一个应用机器人发起或加入该日程会议 |
| `bot is not host` | 应用机器人已在会中，但不是当前 Host | 先把主持人转移给应用机器人，或由当前 Host/Owner 结束会议 |
| `switch for allowing agents to join meetings is disabled` | 会议未开启会中智能体/Agent 能力，或会议 Owner 未命中对应发布门禁 | 确认日程会前 AI/Agent 设置与发布门禁后重试 |

## 参考

- [lark-vc-agent-meeting-start](lark-vc-agent-meeting-start.md) — 启动并加入会议
- [lark-vc-agent-meeting-invite](lark-vc-agent-meeting-invite.md) — 会中邀请
- [lark-vc-agent-meeting-leave](lark-vc-agent-meeting-leave.md) — 仅让机器人离会
