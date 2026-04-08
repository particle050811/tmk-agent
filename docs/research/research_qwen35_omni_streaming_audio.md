# Qwen3.5-Omni 对语音流式输入输出的支持调研

## 结论

结论分开说：

1. `Qwen3.5-Omni` 支持音频输入，也支持音频输出。
2. 如果是普通多模态调用，它支持上传音频作为输入，并可返回文本或音频结果。
3. 如果要实现“持续监听电脑麦克风、边说边收、边翻译边播”的实时体验，应该优先看 `Qwen-Omni-Realtime` 的 WebSocket API，而不是只看普通推理接口。
4. 所以答案是：`支持语音流式输入输出，但要区分普通 Omni 接口和 Realtime 接口。`

## 能力拆解

### 1. 音频输入

Qwen Omni 文档明确支持音频作为输入模态。

这意味着你可以：

1. 传入本地语音文件；
2. 传入 base64 音频；
3. 在某些实时接口中连续发送音频分片。

### 2. 音频输出

文档中也提供了音频生成能力，可以把模型响应转成语音。

这对本题的加分项 `TTS 输出` 很直接。

### 3. 流式输出

在普通请求模式下，模型支持流式返回文本或音频片段。

### 4. 真正的实时语音对话

阿里云 Model Studio 的 `Realtime API` 提供了基于 `WebSocket` 的实时会话机制，支持：

1. 持续追加音频输入；
2. 服务端 VAD；
3. 流式文本或音频响应；
4. 中断、会话控制、事件驱动处理。

这更接近 Mini TMK Agent 的“持续监听并触发翻译”需求。

## 对本题的意义

如果你的目标是 CLI 里的“流式同传模式”，Qwen 系列可以有两种接法：

### 方案 A：麦克风分段 -> 普通 Omni/ASR 接口

```text
Mic -> VAD -> 切片音频 -> 调用模型识别/翻译 -> 输出文本
```

优点：

1. 实现简单；
2. 易于调试；
3. 对 CLI MVP 足够。

缺点：

1. 实时性不如 Realtime；
2. 说话中途的增量反馈可能较弱；
3. 端到端延迟依赖切片长度。

### 方案 B：麦克风流 -> Qwen Realtime WebSocket

```text
Mic -> PCM chunk -> WebSocket send
                  -> server VAD / realtime session
                  -> partial/final transcript
                  -> translated text / audio response
```

优点：

1. 更接近真实同传；
2. 流式体验更自然；
3. 服务端已经提供部分实时会话能力。

缺点：

1. 代码复杂度更高；
2. 要处理事件协议、重连、缓冲、session；
3. CLI 中的音频播放和并发控制会更复杂。

## 是否“开箱即用”

不算完全开箱即用，原因是：

1. 你仍然要自己采集本地麦克风；
2. 你仍然要做音频编码和 chunk 发送；
3. 你仍然要处理终端 UI、并发、取消、重试；
4. 如果不是直接做端到端翻译，还要自行组织 `ASR -> Translation` 的链路。

但模型能力本身是具备的。

## 推荐判断

针对这道题，我建议：

1. MVP 先做 `麦克风 + VAD + chunked ASR + 文本翻译`；
2. 如果时间够，再升级到 `Realtime API`；
3. 如果要做加分项 TTS，再接音频输出链路。

原因很简单：CLI 场景先把可靠性做出来，比“一上来就做全双工实时语音”更重要。

## Go 接入示例

下面是一个“持续发送音频 chunk 到实时会话”的伪代码示例，重点是结构，不绑定具体 SDK。

```go
package main

import (
	"encoding/base64"
	"encoding/json"
)

type InputAudioAppendEvent struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}

func buildAppendEvent(pcm []byte) ([]byte, error) {
	ev := InputAudioAppendEvent{
		Type:  "input_audio_buffer.append",
		Audio: base64.StdEncoding.EncodeToString(pcm),
	}
	return json.Marshal(ev)
}
```

如果采用普通的“分段识别”模式，代码通常会更简单：

```go
func shouldSendChunk(samples []int16, isSpeech bool) bool {
	if !isSpeech {
		return false
	}
	return len(samples) >= 16000 // 例如累计 1 秒 16kHz 单声道 PCM
}
```

## 重要工程点

### 1. 不要把“支持音频输入输出”误解为“自动持续监听”

模型支持多模态，不等于 SDK 自动帮你完成：

1. 采集麦克风；
2. VAD；
3. 音频缓冲；
4. 断句；
5. 本地播放。

这些仍是客户端职责。

### 2. Realtime 比普通接口更贴近需求

如果题目强调“持续监听”和“流式翻译”，`Realtime API` 的匹配度明显更高。

### 3. 服务端 VAD 可以降低客户端复杂度

Realtime 文档里有 `turn_detection`/`server_vad` 配置，这意味着你不一定需要把所有断句逻辑都放在本地做。

但本地依然建议保留一个轻量门控，用于：

1. 降噪；
2. 避免背景噪音持续上传；
3. 降低 token / 带宽 / API 成本。

## 最终建议

对这道题的判断是：

1. `Qwen3.5-Omni 可以作为候选模型`；
2. `如果追求真正的流式语音输入输出，应优先评估 Qwen Omni Realtime API`；
3. `如果追求稳定交付，先做分段式流转，再逐步升级到 Realtime。`

## 参考链接

1. Qwen Omni 文档: https://www.alibabacloud.com/help/en/model-studio/qwen-omni
2. Qwen Realtime API 文档: https://www.alibabacloud.com/help/en/model-studio/realtime
3. DashScope / Model Studio OpenAI-compatible 说明: https://www.alibabacloud.com/help/en/model-studio/

## 备注

这里有一个重要区分：

1. `Qwen3.5-Omni` 指模型能力；
2. `Realtime API` 指会话协议和实时流式交互能力。

实际工程里两者通常要结合起来看。
