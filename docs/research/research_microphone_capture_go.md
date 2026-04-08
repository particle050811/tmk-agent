# Go 实现持续监听电脑麦克风的库调研

## 结论

如果目标是用 Go 在 Linux/macOS 上做 CLI 版持续监听，优先级建议如下：

1. 首选 `malgo`
2. 备选 `portaudio`
3. 播放输出可单独考虑 `oto`

## 库对比

## 1. malgo

仓库：`gen2brain/malgo`

### 优点

1. Go 封装的是 `miniaudio`；
2. 支持 capture / playback / duplex；
3. 跨平台；
4. 对实时音频应用比较友好；
5. 一般不需要额外系统级大型依赖，工程集成相对直接。

### 适合本题的原因

1. Mini TMK Agent 需要持续从麦克风抓取 PCM；
2. 后续可能还要做 TTS 播放；
3. `malgo` 比较适合做“一进一出”的底层音频设备层。

## 2. portaudio

仓库：`gordonklaus/portaudio`

### 优点

1. 历史悠久；
2. 生态成熟；
3. 文档和范例较多；
4. 很适合做基础的音频采集与播放。

### 缺点

1. 经常需要系统先安装 PortAudio；
2. 不同平台的环境准备可能比 `malgo` 更麻烦；
3. 对“开箱即用”要求高的作业项目不一定最优。

## 3. oto

仓库：`ebitengine/oto`

### 用途

`oto` 更偏向音频播放，不是主要的麦克风采集库。

如果你做加分项 TTS，可以考虑：

1. 输入采集用 `malgo`；
2. 输出播放用 `oto` 或继续用 `malgo`。

## 推荐方案

### MVP 推荐

`malgo + webrtcvad + 外部模型 API`

理由：

1. 结构简单；
2. 能覆盖持续监听；
3. 后续扩展到播放也方便；
4. 更适合做跨平台 CLI。

### 保守方案

`portaudio + webrtcvad + 外部模型 API`

如果你已经熟悉 PortAudio，也能做，但环境配置体验通常略差。

## 典型音频参数

建议从下面的 PCM 格式起步：

1. `16kHz`
2. `mono`
3. `16-bit signed PCM`

这是很多 ASR / VAD 工具的常见兼容格式，也方便后面送给模型。

## Go 代码示例

下面给一个基于 `malgo` 的最小采集结构示例。

```go
package main

import (
	"log"

	"github.com/gen2brain/malgo"
)

func main() {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		log.Println(message)
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 16000
	deviceConfig.Alsa.NoMMap = 1

	onRecvFrames := func(outputSamples, inputSamples []byte, frameCount uint32) {
		_ = outputSamples
		_ = frameCount

		// inputSamples 即实时麦克风 PCM 数据
		// 这里通常会送到 ring buffer / VAD / encoder
		_ = inputSamples
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		log.Fatal(err)
	}

	select {}
}
```

如果采用 `portaudio`，结构通常也类似：

```go
package main

import "github.com/gordonklaus/portaudio"

func main() {
	portaudio.Initialize()
	defer portaudio.Terminate()

	in := make([]int16, 512)
	stream, _ := portaudio.OpenDefaultStream(1, 0, 16000, len(in), &in)
	defer stream.Close()
	_ = stream.Start()

	for {
		_ = stream.Read()
		// in 中是采集到的 PCM 样本
	}
}
```

## 工程建议

### 1. 增加 ring buffer

不要在音频回调里直接做网络请求或复杂模型调用。

推荐：

1. 回调线程只负责把 PCM 写入 ring buffer；
2. 后台 goroutine 负责 VAD、切片、上传；
3. 避免阻塞采集线程。

### 2. 统一内部音频格式

建议内部统一成：

1. `PCM s16le`
2. `16kHz`
3. `mono`

这样 VAD、ASR、文件转录都能共享代码。

### 3. 设备异常要可恢复

需要考虑：

1. 麦克风设备不存在；
2. 权限不足；
3. 热插拔；
4. 设备被其他程序占用。

CLI 模式里至少要给出明确报错。

## 最终建议

如果让我现在为这道题选型，我会用：

1. `malgo` 负责持续监听麦克风；
2. `webrtcvad` 负责语音活动检测；
3. `Qwen Realtime` 或 `ASR + 翻译模型` 负责识别与翻译；
4. 如需播报，再接 `oto` 或继续使用 `malgo` 播放。

## 参考链接

1. malgo: https://github.com/gen2brain/malgo
2. miniaudio: https://miniaud.io/
3. portaudio-go: https://github.com/gordonklaus/portaudio
4. PortAudio 文档: https://portaudio.github.io/docs.html
5. oto: https://github.com/ebitengine/oto
