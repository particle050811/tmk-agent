# Eino 框架对语音流式输入输出的支持调研

## 结论

结论先说：

1. `Eino` 目前没有面向“麦克风输入 / ASR / TTS / 音频流设备”的原生高层组件。
2. `Eino` 有很强的 `ChatModel`、`Lambda`、`Graph`、`Stream` 编排能力，适合把“音频采集 -> ASR -> 翻译 -> TTS”串成一条流水线。
3. 因此它更适合作为 `Agent/Workflow orchestration layer`，而不是直接承担音频 I/O 的框架。
4. 如果要做 Mini TMK Agent，合理方案是：`Eino 负责编排`，`外部音频库负责采集/播放`，`外部模型 API 负责 ASR/TTS/翻译`。

## 依据

从 Eino 的官方概览和组件设计看，核心能力聚焦在：

1. LLM 调用与抽象；
2. Prompt、Tool、Retriever、Memory 等 AI 组件；
3. Graph/Chain 形式的编排；
4. Stream 结果处理。

公开资料中没有看到专门的：

1. `Microphone` 输入组件；
2. `AudioFrame` 流式抽象；
3. `ASR` 标准接口；
4. `TTS` 标准接口；
5. 面向实时语音会话的内建 session/runtime。

这意味着“是否支持语音流式输入输出”的答案要分两层理解：

### 1. 框架层

`支持流式编排`，但不是“开箱即用的语音框架”。

### 2. 业务落地层

可以支持，但需要你自己补齐下面这些模块：

1. 麦克风采集；
2. VAD/静音检测；
3. 音频 chunk 缓冲与切片；
4. ASR 或 Realtime 语音模型接入；
5. 可选的 TTS 播放。

## 适合本题的使用方式

推荐把 Eino 放在“编排层”，不要让它负责底层音频设备操作。

一个可落地的执行流如下：

```text
Mic Capture
  -> VAD / Speech Gate
  -> ASR (streaming or chunked)
  -> Eino Lambda/Graph
       -> translation
       -> post-process / transcript formatting
  -> Terminal output
  -> Optional TTS
```

## 推荐架构

### 方案 A：Eino 只负责文本链路

```text
麦克风 -> Go 音频库 -> VAD -> ASR
                         -> 源语言文本
                         -> Eino 调用翻译模型
                         -> 目标语言文本
```

优点：

1. 解耦清晰；
2. 易于测试；
3. Eino 的价值点明确，适合做翻译链和后处理；
4. 出问题时更容易定位是音频层还是模型层。

### 方案 B：Eino 编排整个语音链路

可以通过 `Lambda` 节点包装外部 ASR/TTS SDK，把它们纳入 Graph。

优点：

1. 统一链路编排；
2. 便于后面扩展 transcript、summarize、tool call。

缺点：

1. 音频实时链路不一定天然适合放进通用 Agent 图编排；
2. 调试复杂度更高；
3. 对 CLI 版 MVP 来说偏重。

## 判断

如果目标是 3 天内做出一个稳定可演示的 CLI MVP，我的建议是：

1. `不要依赖 Eino 提供原生语音流能力`，因为它并没有现成的音频 I/O 设施；
2. `把 Eino 用在翻译和流程编排上`，这是它的强项；
3. `实时语音链路单独实现`，这样工程风险最低。

## Go 侧接入示例

下面示例演示如何把外部语音识别结果送入 Eino 的文本处理节点。

```go
package main

import (
	"context"
	"fmt"
)

type ASRChunk struct {
	Text     string
	IsFinal  bool
	Language string
}

type Translator interface {
	Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error)
}

func handleASRChunk(ctx context.Context, tr Translator, chunk ASRChunk, targetLang string) error {
	if chunk.Text == "" {
		return nil
	}

	translated, err := tr.Translate(ctx, chunk.Text, chunk.Language, targetLang)
	if err != nil {
		return err
	}

	fmt.Printf("SRC: %s\n", chunk.Text)
	fmt.Printf("DST: %s\n", translated)
	return nil
}
```

这个模式的关键点是：音频相关逻辑不强绑在 Eino 内部，Eino 只接收“已经转成文本的增量结果”。

## 风险与限制

1. 如果你坚持“纯 Eino 完成端到端语音流”，会发现大量工作实际上要自己补；
2. 语音实时场景对延迟和缓存控制敏感，通用 Agent 编排层不是主要瓶颈优化点；
3. 对本题来说，先保证“持续监听 + 增量转写 + 增量翻译”比追求框架统一更重要。

## 最终建议

对于 Mini TMK Agent：

1. `Eino 可以用`；
2. `但不要把它当成语音 SDK`；
3. 最佳定位是：`文本翻译编排层 / Agent 工作流层`；
4. 麦克风、VAD、ASR、TTS 仍需额外库或外部模型 API 支撑。

## 参考链接

1. Eino GitHub: https://github.com/cloudwego/eino
2. Eino 官方文档概览: https://www.cloudwego.io/zh/docs/eino/overview/
3. Eino 组件文档: https://www.cloudwego.io/zh/docs/eino/core_modules/components/

## 备注

“Eino 不支持语音流式输入输出”的说法更准确地应表达为：

`它没有现成的原生语音 I/O 能力，但可以用于编排语音应用的上层工作流。`
