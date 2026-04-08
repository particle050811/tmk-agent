package streaming

type Chunker struct {
	targetBytes int
	buf         []byte
}

func NewChunker(sampleRate int, channels int, chunkMillis int) *Chunker {
	bytesPerSample := 2
	targetBytes := sampleRate * channels * bytesPerSample * chunkMillis / 1000
	if targetBytes <= 0 {
		targetBytes = sampleRate * channels * bytesPerSample / 10
	}

	return &Chunker{targetBytes: targetBytes}
}

func (c *Chunker) Push(in []byte) [][]byte {
	if len(in) == 0 {
		return nil
	}

	c.buf = append(c.buf, in...)
	var chunks [][]byte

	for len(c.buf) >= c.targetBytes {
		chunk := make([]byte, c.targetBytes)
		copy(chunk, c.buf[:c.targetBytes])
		chunks = append(chunks, chunk)
		c.buf = c.buf[c.targetBytes:]
	}

	return chunks
}
