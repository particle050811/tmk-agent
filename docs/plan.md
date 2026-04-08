# Mini TMK Agent 实施计划

## 当前技术决策

基于当前约束，项目采用下面的技术路线：

1. 不使用 `Eino`
2. 使用 `malgo` 负责本地麦克风持续采集
3. 使用 `Qwen Omni Realtime API` 负责实时语音输入输出
4. 暂时不实现本地 `trigger policy`
5. 先优先完成 CLI 的 `stream` 模式，再补 `transcript` 模式

这里的核心判断很直接：

1. 这道题的关键是做出稳定的语音实时链路；
2. 当前不需要引入额外编排框架；
3. `malgo + realtime websocket` 是最短实现路径；
4. 暂不做 trigger policy，可以减少本地状态机和 VAD 复杂度。

## MVP 范围

第一阶段只做最小可演示版本。

### 目标能力

1. CLI 启动 `stream` 模式
2. 持续监听电脑麦克风
3. 将 PCM 音频分片持续发送到 Qwen Realtime 会话
4. 接收服务端返回的实时识别文本和翻译文本
5. 在终端持续打印增量结果和最终结果

### 暂不实现

1. 本地 trigger policy
2. 本地 VAD
3. 本地 TTS 播放
4. Eino 编排
5. 复杂 UI
6. 双端 RTC 通信

## 系统架构

```text
mini-tmk-agent stream
  -> malgo microphone capture
  -> PCM chunk channel
  -> websocket client
  -> Qwen Omni Realtime session
  -> partial/final transcript events
  -> translated text events
  -> terminal renderer
```

## 模块拆分

建议按下面结构实现：

```text
cmd/mini-tmk-agent/
internal/config/
internal/audio/
internal/realtime/
internal/streaming/
internal/render/
```

模块职责如下：

### `cmd/mini-tmk-agent`

负责：

1. CLI 参数解析
2. 启动 `stream` / `transcript`
3. 组装依赖

### `internal/config`

负责：

1. 读取环境变量
2. API Key / Base URL / Model 配置
3. 音频参数配置

### `internal/audio`

负责：

1. 初始化 `malgo`
2. 持续采集麦克风 PCM
3. 将音频数据写入 channel / ring buffer

### `internal/realtime`

负责：

1. 建立 WebSocket 连接
2. 发送 session 配置
3. 发送 `input_audio_buffer.append`
4. 接收并解析实时事件
5. 处理关闭、错误、重连

### `internal/streaming`

负责：

1. 把采集到的 PCM 按固定 chunk 聚合
2. 控制发送节奏
3. 协调音频发送和事件接收

### `internal/render`

负责：

1. 渲染 partial transcript
2. 渲染 final transcript
3. 渲染 translated text
4. 渲染错误和连接状态

## 建议目录

```text
cmd/mini-tmk-agent/main.go
internal/config/config.go
internal/audio/microphone.go
internal/realtime/client.go
internal/realtime/events.go
internal/streaming/session.go
internal/render/terminal.go
```

## 数据流设计

在当前阶段，不做本地 trigger policy，音频链路可以尽量简单。

### 发送链路

```text
malgo callback
  -> append PCM bytes into buffered channel
  -> streaming session aggregates fixed-size chunk
  -> websocket sends input_audio_buffer.append
```

### 接收链路

```text
websocket read loop
  -> parse JSON event
  -> map to internal event struct
  -> render to terminal
```

## 核心并发模型

建议至少拆成 4 个 goroutine：

1. 麦克风采集 goroutine
2. 音频 chunk 发送 goroutine
3. WebSocket 接收 goroutine
4. 主控制 goroutine

这样可以把设备采集、网络发送和终端渲染隔离开，避免互相阻塞。

## 核心代码骨架

下面的代码不是完整实现，而是当前方案下最值得先落盘的核心骨架。

## 1. 配置结构

```go
package config

import "os"

type Config struct {
	APIKey      string
	BaseURL     string
	Model       string
	SampleRate  uint32
	Channels    uint32
	ChunkMillis int
}

func Load() Config {
	return Config{
		APIKey:      os.Getenv("DASHSCOPE_API_KEY"),
		BaseURL:     getenv("QWEN_REALTIME_BASE_URL", "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"),
		Model:       getenv("QWEN_REALTIME_MODEL", "qwen-omni-realtime"),
		SampleRate:  16000,
		Channels:    1,
		ChunkMillis: 200,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

## 2. `malgo` 持续采集

```go
package audio

import (
	"context"

	"github.com/gen2brain/malgo"
)

type Microphone struct {
	ctx    *malgo.AllocatedContext
	device *malgo.Device
}

func StartMicrophone(ctx context.Context, sampleRate uint32, channels uint32, out chan<- []byte) (*Microphone, error) {
	maCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatS16
	cfg.Capture.Channels = channels
	cfg.SampleRate = sampleRate
	cfg.Alsa.NoMMap = 1

	callbacks := malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, frameCount uint32) {
			_ = outputSamples
			_ = frameCount

			buf := make([]byte, len(inputSamples))
			copy(buf, inputSamples)

			select {
			case out <- buf:
			default:
			}
		},
	}

	device, err := malgo.InitDevice(maCtx.Context, cfg, callbacks)
	if err != nil {
		_ = maCtx.Uninit()
		maCtx.Free()
		return nil, err
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		_ = maCtx.Uninit()
		maCtx.Free()
		return nil, err
	}

	m := &Microphone{
		ctx:    maCtx,
		device: device,
	}

	go func() {
		<-ctx.Done()
		m.Close()
	}()

	return m, nil
}

func (m *Microphone) Close() {
	if m.device != nil {
		m.device.Uninit()
	}
	if m.ctx != nil {
		_ = m.ctx.Uninit()
		m.ctx.Free()
	}
}
```

## 3. Realtime 事件结构

```go
package realtime

type SessionUpdateEvent struct {
	Type    string         `json:"type"`
	Session SessionPayload `json:"session"`
}

type SessionPayload struct {
	Model string `json:"model"`
}

type InputAudioAppendEvent struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}

type GenericEvent struct {
	Type string `json:"type"`
}
```

## 4. WebSocket 客户端

```go
package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/coder/websocket"
)

type Client struct {
	conn *websocket.Conn
}

func Dial(ctx context.Context, url string, apiKey string) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: map[string][]string{
			"Authorization": {"Bearer " + apiKey},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

func (c *Client) SendSessionUpdate(ctx context.Context, model string) error {
	ev := SessionUpdateEvent{
		Type: "session.update",
		Session: SessionPayload{
			Model: model,
		},
	}
	return c.writeJSON(ctx, ev)
}

func (c *Client) AppendAudio(ctx context.Context, pcm []byte) error {
	ev := InputAudioAppendEvent{
		Type:  "input_audio_buffer.append",
		Audio: base64.StdEncoding.EncodeToString(pcm),
	}
	return c.writeJSON(ctx, ev)
}

func (c *Client) writeJSON(ctx context.Context, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.conn.Write(ctx, websocket.MessageText, b)
}
```

## 5. 固定 chunk 聚合器

不做 trigger policy 时，核心就是“按固定时长切包并持续发”。

```go
package streaming

type Chunker struct {
	targetBytes int
	buf         []byte
}

func NewChunker(sampleRate int, channels int, chunkMillis int) *Chunker {
	bytesPerSample := 2
	target := sampleRate * channels * bytesPerSample * chunkMillis / 1000
	return &Chunker{targetBytes: target}
}

func (c *Chunker) Push(in []byte) (chunks [][]byte) {
	c.buf = append(c.buf, in...)

	for len(c.buf) >= c.targetBytes {
		chunk := make([]byte, c.targetBytes)
		copy(chunk, c.buf[:c.targetBytes])
		chunks = append(chunks, chunk)
		c.buf = c.buf[c.targetBytes:]
	}

	return chunks
}
```

## 6. 主流程

```go
package streaming

import (
	"context"
)

type AudioSender interface {
	AppendAudio(ctx context.Context, pcm []byte) error
}

func Run(ctx context.Context, audioIn <-chan []byte, sender AudioSender, sampleRate int, channels int, chunkMillis int) error {
	chunker := NewChunker(sampleRate, channels, chunkMillis)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pcm := <-audioIn:
			for _, chunk := range chunker.Push(pcm) {
				if err := sender.AppendAudio(ctx, chunk); err != nil {
					return err
				}
			}
		}
	}
}
```

## 7. 终端输出

```go
package render

import "fmt"

func PrintStatus(status string) {
	fmt.Printf("[status] %s\n", status)
}

func PrintSource(text string) {
	fmt.Printf("[src] %s\n", text)
}

func PrintTarget(text string) {
	fmt.Printf("[dst] %s\n", text)
}
```

## 开发顺序

建议按下面顺序推进，不要同时铺太多面。

### 第 1 步

搭 CLI 骨架：

1. `mini-tmk-agent stream`
2. 参数解析
3. 环境变量读取

### 第 2 步

打通本地音频采集：

1. 接入 `malgo`
2. 终端打印采集状态
3. 验证 PCM 能持续输出

### 第 3 步

打通 Realtime 连接：

1. 建立 WebSocket
2. 发送 `session.update`
3. 确认服务端能回事件

### 第 4 步

打通音频上行：

1. 增加 chunk 聚合
2. 持续发送 `input_audio_buffer.append`
3. 验证服务端能接收音频

### 第 5 步

打通结果下行：

1. 解析 transcript / response 事件
2. 区分 partial 和 final
3. 终端渲染源文本和目标文本

### 第 6 步

补工程质量：

1. 错误处理
2. 优雅退出
3. README
4. 基础测试

## 测试计划

当前阶段的测试重点不在“算法”，而在“链路”。

### 单元测试

1. `config.Load`
2. `Chunker.Push`
3. Realtime 事件序列化/反序列化

### 集成测试

1. 本地麦克风设备可打开
2. WebSocket 可连接
3. 音频 chunk 可发送
4. 收到服务端事件后可正确打印

## 风险点

当前方案的主要风险如下：

1. 不做 trigger policy 时，会持续上传环境音，成本和噪音都更高
2. Realtime API 事件协议需要按实际文档对齐，字段名可能需要细调
3. 终端里处理 partial/final 文本时，输出刷新策略需要设计清楚
4. 麦克风权限和设备兼容性可能影响演示

## 后续演进

在 MVP 跑通之后，再按下面顺序增强：

1. 增加本地 trigger policy
2. 增加 transcript 模式
3. 增加 TTS 播放
4. 增加配置文件
5. 增加 Web UI

## 当前推荐结论

这个阶段最合适的做法不是继续发散调研，而是直接按下面的最小方案开工：

1. `malgo` 负责麦克风采集
2. `coder/websocket` 负责 Realtime WebSocket
3. `Qwen Omni Realtime API` 负责语音输入输出
4. 先做固定 chunk 发送
5. 暂时不做 trigger policy

这条路径最短，也最符合你现在给定的约束。
