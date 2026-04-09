# tmk-agent

这个 README 只记录项目常见启动命令格式，方便直接复制使用。

## 1. 准备环境变量

先准备 `.env`：

```bash
cp .env.example .env
```

至少要填写：

```env
DASHSCOPE_API_KEY=your_dashscope_api_key
```

程序启动时会自动读取项目根目录下的 `.env`。

## 2. 常见启动方式

### 2.1 直接运行源码

```bash
go run . <subcommand> [flags]
```

可用子命令：

- `stream`
- `transcript`

### 2.2 先编译再运行

编译：

```bash
go build -o tmk-agent .
```

运行：

```bash
./tmk-agent <subcommand> [flags]
```

## 3. 实时流式翻译

命令格式：

```bash
go run . stream --source-lang <源语言> --target-lang <目标语言>
```

示例：

```bash
go run . stream --source-lang zh --target-lang en
```

```bash
go run . stream --source-lang en --target-lang zh
```

如果是编译后的可执行文件：

```bash
./tmk-agent stream --source-lang zh --target-lang en
```

说明：

- `--source-lang` 默认值是 `zh`
- `--target-lang` 默认值是 `en`
- 启动后会使用麦克风采集音频

## 4. 音频文件转字幕

命令格式：

```bash
go run . transcript --file <音频文件> --output <输出srt文件> --source-lang <源语言> --target-lang <目标语言>
```

示例：

```bash
go run . transcript --file ./test/test.mp3 --output ./out/test.srt --source-lang zh --target-lang en
```

```bash
go run . transcript --file ./test/test.wav --output ./out/test.srt --source-lang en --target-lang zh
```

如果是编译后的可执行文件：

```bash
./tmk-agent transcript --file ./test/test.mp3 --output ./out/test.srt --source-lang zh --target-lang en
```

说明：

- `--file` 必填
- `--output` 必填
- `--source-lang` 必填
- `--target-lang` 必填
- 支持的音频格式见当前实现，常用可按 `mp3`、`wav` 理解

## 5. 常用环境变量

常见可调项：

```env
QWEN_REALTIME_BASE_URL=wss://dashscope.aliyuncs.com/api-ws/v1/realtime
QWEN_REALTIME_MODEL=qwen3-omni-flash-realtime
QWEN_TRANSCRIPT_MODEL=qwen3-omni-flash
TMK_SAMPLE_RATE=16000
TMK_CHANNELS=1
TMK_CHUNK_MILLIS=200
TMK_AUDIO_BUFFER_FRAMES=64
TMK_AUDIO_DEVICE=
TMK_DEBUG=false
TMK_DEBUG_AUDIO_DIR=./tmp/debug-audio
TMK_DEBUG_AUDIO_SECONDS=15
```

其中：

- `TMK_AUDIO_DEVICE` 可按设备名部分匹配来指定录音设备
- `TMK_DEBUG=true` 可打开调试输出
- `TMK_DEBUG_AUDIO_DIR` 配置后会输出调试音频文件
