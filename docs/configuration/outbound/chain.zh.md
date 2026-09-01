### 结构

```json
{
  "type": "chain",
  "tag": "chain-out",
  "outbounds": [
    "proxy-a",
    "proxy-b",
    "proxy-c"
  ]
}
```

### 字段

#### outbounds

==必填==

组成链式代理的 outbound 标签列表，顺序为数据包路径顺序。

- `outbounds[0]` 为第一跳（最靠近客户端）。
- 最后一项为最终出口。
- 最少 1 个（单跳通常无必要），推荐至少 2 个。

### 行为说明

Chain 是一等公民 Outbound，使多跳代理配置显式且对用户友好。

在配置加载阶段，编译器会：

1. 验证每个跳点存在且不是另一个 chain（当前禁止嵌套）。
2. 拒绝已设置 `detour` 的跳点，以及将 `direct` 用作非最终跳点的情况。
3. 自动创建带正确 detour 的私有中间 outbound。
4. 将链式入口（第一跳）以 chain 自身的 tag 暴露。

原始 outbound 保持不变，仍可独立使用。流量策略继续完全由路由规则控制。

### 限制（Fail-Closed）

- 不支持嵌套 chain。
- 已包含 `detour` 的跳点不能用于 chain。
- `direct` 不能作为非最终跳点。
- 内部合成 tag 被保留；与用户标签冲突时会在加载阶段明确报错。

### 示例

```json
{
  "outbounds": [
    {
      "type": "shadowsocks",
      "tag": "ss-entry",
      "server": "entry.example.com",
      "server_port": 8388,
      "method": "aes-256-gcm",
      "password": "password"
    },
    {
      "type": "vless",
      "tag": "vless-exit",
      "server": "exit.example.com",
      "server_port": 443,
      "uuid": "uuid",
      "tls": { "enabled": true, "server_name": "exit.example.com" }
    },
    {
      "type": "chain",
      "tag": "my-chain",
      "outbounds": ["ss-entry", "vless-exit"]
    },
    {
      "type": "direct",
      "tag": "direct"
    }
  ],
  "route": {
    "final": "my-chain"
  }
}
```
