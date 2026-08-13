# `vc +meeting-screenshot`

获取正在进行且已启用录制的会议当前最终混流截图，并保存为 JPEG。

```bash
lark-cli vc +meeting-screenshot --as user --meeting-id <long_meeting_id>
lark-cli vc +meeting-screenshot --as bot --meeting-id <long_meeting_id> --output ./current.jpg --overwrite
```

- `--meeting-id` 必须是长数字会议 ID，不接受 9 位会议号。
- 身份必须沿用发现该 `meeting_id` 时使用的身份；用户身份需要当前用户在会中，应用身份需要机器人具备会中读取权限。
- 会议必须仍在进行中并且已开启录制。截图固定为最终混流画面，不支持调用方选择参会人或共享内容。
- 默认写入 `.lark-vc/screenshots/<meeting_id>-<UTC timestamp>.jpg`；已有目标文件时默认失败，只有 `--overwrite` 可以替换。
- 成功输出包含绝对文件路径、字节数、JPEG content type、SHA-256 和服务端 `log_id`。失败不会替换已有文件。
