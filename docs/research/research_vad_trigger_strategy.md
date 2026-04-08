# 有意义语音输入检测与流式翻译触发策略调研

## 结论

不要用“音量大于阈值”这种单一策略来决定是否开始翻译。

对 Mini TMK Agent，更可靠的方案是：

`能量门限 + WebRTC VAD + 最小时长 + 结束静音 + 可选 ASR 二次确认`

这是一个兼顾实现成本、准确性和实时性的组合。

## 为什么不能只看音量

背景噪音、键盘声、风扇声、碰撞声都可能有较高能量。

如果只用 RMS/峰值阈值，会出现：

1. 经常误触发；
2. 持续上传无意义音频；
3. 模型成本升高；
4. 终端输出大量垃圾文本。

## 推荐策略

## 第一层：能量门控

先做一个极轻量的能量过滤。

例如：

1. 计算 10ms 或 20ms 帧的 RMS；
2. 低于阈值直接丢弃；
3. 只有超过阈值的帧才送入 VAD。

作用：

1. 降低 VAD 调用量；
2. 滤掉绝大多数安静背景；
3. 减少误判。

## 第二层：WebRTC VAD

使用 WebRTC VAD 判断帧是否像“人声”。

这类 VAD 的优点是：

1. 轻量；
2. 实时；
3. 工程里已被大量验证；
4. 对 10/20/30ms PCM 帧处理很方便。

在 Go 里可以直接用：

1. `github.com/maxhawkins/go-webrtcvad`

该库对应 WebRTC VAD，支持不同 aggressiveness 模式。

## 第三层：起始触发条件

不要一帧判定为人声就立刻开始翻译。

建议要求满足以下条件之一：

1. 连续 `N` 帧被判定为 speech；
2. 或最近 `M` 帧中 speech 占比超过阈值。

例如：

1. 20ms 一帧；
2. 连续 200ms 以上为 speech 才进入“讲话中”状态。

这样可以过滤掉短暂敲击声和瞬时噪音。

## 第四层：结束判定

进入“讲话中”后，不要一出现一帧静音就立刻提交。

建议增加 `hangover / end-of-speech silence`：

1. 连续 500ms 到 800ms 静音，才判定一句话结束；
2. 在结束前保留少量尾音缓存，避免截断。

这样对中文口语里的短暂停顿更稳。

## 第五层：最小语音片段长度

即使通过了 VAD，也不要提交太短的语音段。

建议：

1. 小于 300ms 到 500ms 的片段默认丢弃；
2. 或者只作为“继续缓冲”的信号，不立即发请求。

## 第六层：ASR 二次确认

如果你走的是“分段识别 -> 翻译”的链路，可以在提交前做一次低成本确认：

1. 若 ASR 返回空文本，丢弃；
2. 若文本仅为噪音词、语气词且长度太短，可不翻译；
3. 若置信度很低，也可跳过。

这样可以进一步减少误触发。

## 服务端 VAD 是否够用

如果使用 `Qwen Realtime API`，文档中提供了 `turn_detection` 和 `server_vad`。

这很好用，但我仍然建议本地保留一层轻量门控，原因是：

1. 本地先挡掉纯噪音，可以减少上传；
2. 降低带宽消耗；
3. 降低模型调用成本；
4. 避免服务端不断收到无意义片段。

所以推荐做法是：

`客户端轻量门控 + 服务端 VAD`

而不是二选一。

## 推荐状态机

一个简单有效的状态机如下：

```text
Idle
  -> (energy pass + vad speech ratio enough)
PreSpeech
  -> (speech duration >= threshold)
Speaking
  -> (silence long enough)
Commit
  -> Idle
```

说明：

1. `Idle`：等待输入；
2. `PreSpeech`：疑似讲话开始，但先不立即触发；
3. `Speaking`：正式开始收集并流式发送；
4. `Commit`：一次语句结束，提交最终片段或刷新翻译结果。

## 建议参数

下面是一组适合先跑通 MVP 的默认参数：

1. 帧长：`20ms`
2. 采样率：`16kHz`
3. 声道：`mono`
4. WebRTC VAD 模式：`2`
5. 起始门限：连续 `10` 帧 speech，约 `200ms`
6. 结束门限：连续 `25-40` 帧 silence，约 `500-800ms`
7. 最小片段长度：`400ms`
8. 最大片段长度：`8-15s`

最大片段长度也要限制，否则用户长时间不停说时，缓冲会越来越大，延迟也会持续恶化。

## Go 代码示例

下面是一个简化版思路：

```go
package main

import vad "github.com/maxhawkins/go-webrtcvad"

type Gate struct {
	speechFrames  int
	silenceFrames int
	speaking      bool
}

func (g *Gate) Update(isSpeech bool) (start bool, stop bool) {
	if isSpeech {
		g.speechFrames++
		g.silenceFrames = 0
	} else {
		g.silenceFrames++
		if !g.speaking {
			g.speechFrames = 0
		}
	}

	if !g.speaking && g.speechFrames >= 10 {
		g.speaking = true
		return true, false
	}

	if g.speaking && g.silenceFrames >= 30 {
		g.speaking = false
		g.speechFrames = 0
		g.silenceFrames = 0
		return false, true
	}

	return false, false
}

func classifyFrame(v *vad.VAD, frame []byte, sampleRate int) (bool, error) {
	return v.Process(sampleRate, frame)
}
```

真实工程里还应再加：

1. RMS 预过滤；
2. 环形缓冲区；
3. 起始前回溯缓存；
4. 结束后尾音保留；
5. 取消与超时控制。

## 与流式翻译的结合方式

分两种：

### 方案 A：增量识别，边识别边翻译

1. 进入 `Speaking` 后就开始持续发送 chunk；
2. 收到 partial transcript 后做增量翻译或局部刷新；
3. 收到 final transcript 后再输出最终结果。

### 方案 B：一句话结束后再翻译

1. `Speaking` 阶段只收集；
2. 到 `Commit` 再统一识别和翻译。

方案 A 实时性更强，方案 B 工程更简单。

对这道题的 MVP，我建议：

1. 先做 `句级触发`；
2. 再升级为 `增量触发`。

## 最终建议

最稳妥的落地策略是：

1. 客户端先做 `RMS gate`；
2. 再做 `WebRTC VAD`；
3. 用状态机控制起止；
4. 设置最小讲话时长和结束静音；
5. 若接 Realtime API，再配合 `server_vad`；
6. 若接分段 ASR，再做一次文本有效性确认。

## 参考链接

1. Qwen Realtime API: https://www.alibabacloud.com/help/en/model-studio/realtime
2. go-webrtcvad: https://github.com/maxhawkins/go-webrtcvad
3. WebRTC VAD Python 封装说明（用于理解帧长与采样率限制）: https://github.com/wiseman/py-webrtcvad

## 备注

严格来说，“有意义的输入”不只是“检测到人声”，而是：

`检测到持续的人声，并且这段人声值得触发一次识别/翻译。`

所以必须把 `VAD` 和 `trigger policy` 分开设计。
