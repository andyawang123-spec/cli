# calendar +transfer

把一个日程的**组织者（organizer）**转让给另一个用户或机器人。用户和机器人之间可以任意互转。

## 命令

```bash
# 转让给某人（原组织者保留为参与人）
lark-cli calendar +transfer --event-id <event_id> --to-user-id ou_xxx --yes

# 转让并把原组织者从参与人中移除
lark-cli calendar +transfer --event-id <event_id> --to-user-id ou_xxx --remove-original-organizer --yes

# 指定日历
lark-cli calendar +transfer --calendar-id <calendar_id> --event-id <event_id> --to-user-id ou_xxx --yes

# 重复性日程：必须显式确认整个序列一起转让
lark-cli calendar +transfer --event-id <event_id> --to-user-id ou_xxx --transfer-series --yes

# 预览请求，不实际执行
lark-cli calendar +transfer --event-id <event_id> --to-user-id ou_xxx --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--event-id <id>` | **是** | 日程 ID（`uid_originalTime` 形式） |
| `--to-user-id <ou_...>` | **是** | 接收人 open_id，成为新组织者；用户和机器人都可以 |
| `--calendar-id <id>` | 否 | 日程所在日历 ID（省略则使用主日历） |
| `--remove-original-organizer` | 否 | 转让后把原组织者移出参与人；默认保留。日程在共享日历上时服务端一定会移除 |
| `--transfer-series` | 否 | 确认整个重复性序列一起转让；重复性日程必填 |
| `--yes` | **是**（非 dry-run） | 高敏写操作确认 |
| `--dry-run` | 否 | 预览 API 调用，不执行 |

## 转让方向

转出方和接收方是两个**互相独立**的参数，四种组合都支持：

- **转出方**由 `--as` 决定，必须是日程**当前组织者**的身份。bot 组织的日程用 `--as bot`，用户自己的日程用 `--as user`。用非组织者身份调用会返回 403。
- **接收方**由 `--to-user-id` 决定，传谁的 open_id 就转给谁，是人还是机器人不影响命令写法。

| 方向 | 命令 |
|------|------|
| user → user | `--as user --to-user-id <对方用户 open_id>` |
| user → bot | `--as user --to-user-id <bot 的 open_id>` |
| bot → user | `--as bot --to-user-id <用户 open_id>` |
| bot → bot | `--as bot --to-user-id <另一个 bot 的 open_id>` |

**取接收人 open_id**：

```bash
# 用户
lark-cli contact +search-user --query <姓名> --as user
# 机器人：从它所在群的成员列表里取 bots[] 中的 open_id
lark-cli im +chat-members-list --chat-id <chat_id> --member-types bot
```

机器人的 open_id 同样是 `ou_` 开头；不要传 `cli_` 开头的 app_id，那是应用 ID，不是日程参与人身份。

无论哪个方向，转让都要求转出方和接收方**同租户**，且接收方能通过高管模式的协作校验。

## 重复性日程

后端按 `uid` 定位日程，忽略 `original_time`，**无法只转让某一次实例**。因此传入任何一个实例或例外的 `event_id`，都会把整个序列（含所有例外）一起转让。

是重复性日程且未加 `--transfer-series` 时命令直接失败（`failed_precondition`），不会发出转让请求。收到这个错误时**先向用户确认"整个重复日程都转让"**，得到确认后再带 `--transfer-series` 重跑；不要自动重试。已确认时加 `--transfer-series` 会跳过这次预读。

## 返回中的 `original_organizer_removed`

**共享日历不属于任何组织者，转让时服务端会强制把原组织者移出日程；主日历则会把原组织者保留为参与人。** 转让接口成功时不返回这个结果，所以命令只在能确定时才输出该字段：

| 情况 | 返回 |
|------|------|
| 带 `--remove-original-organizer` | `original_organizer_removed: true` |
| 省略 `--calendar-id`（主日历） | `original_organizer_removed: false` |
| 传了 `--calendar-id` 且未传 `--remove-original-organizer` | **不返回该字段**，stderr 给一条 note 说明共享日历会强制移除 |

字段缺失时**不要**告诉用户"原组织者已保留为参与人"，也不要断言已被移除。需要确认就转让后读一次日程看参与人，或一开始就显式传 `--remove-original-organizer`。

## 提示

- 转让不可逆，且会连同日程上的会议纪要、笔记和附件一起移交给新组织者。
- 需要 `calendar:calendar.event:transfer` 权限；转让前的重复性预读需要 `calendar:calendar.event:read`（带 `--transfer-series` 时不读）。

## 错误码

| code | 含义 | 处理方式 |
|------|------|----------|
| `14` | `--event-id` 格式非法，具体原因在响应的 `details.reason` 里 | `event_id` 必须是 `uid_originalTime` 形式，从 `+get` / `+agenda` 的输出里取，不要手拼 |
| `2` | `--to-user-id` 解析不出用户 | 确认传的是 `ou_` 开头的 open_id，且该用户或机器人在当前租户内 |
| `1001` | `--calendar-id` 解析不出日历 | 用 `lark-cli calendar calendars list` 确认日历 ID；省略该参数时默认主日历 |
| `1002` | 当前身份对该日历没有编辑权限（需要 WRITER 或 OWNER） | 换成日程组织者身份（`--as`），或让日历所有者授予编辑权限 |
| `1003` | 日历已被删除 | 无需转让，直接结束 |
| `1004` | 该日历类型不支持转让 | 只有主日历和共享日历上的日程能转让；会议室、邮箱、Google/Exchange 导入的日历不行 |
| `3001` | 日程不存在 | 确认 `event_id` 与 `calendar_id` 配套，日程未被删除 |
| `3003` | 日程已被删除 | 无需转让，直接结束 |
| `3108` | 接收人已经是该日程的组织者 | 无需转让，直接结束 |
| `3109` | 高管模式下当前身份无权邀请接收人 | 换一个接收人，或让对方主动发起 |
| `3110` | 日程不在组织者的日历上 | 用日程组织者所在的日历 ID 重试；`--calendar-id` 省略时默认主日历，共享日历上的日程要显式传 |
| `3111` | 接收人与原组织者不同租户 | 跨租户不支持转让，只能改为邀请对方为参与人 |
| `3` | 服务端未预期错误 | 可重试一次；持续失败则上报 |

## 参考

- [lark-calendar](../SKILL.md) -- skill 入口与路由
- [重复性日程操作规范](lark-calendar-recurring.md)
