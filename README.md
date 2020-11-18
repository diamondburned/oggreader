# oggreader

A tiny Ogg packet decoder.

## Usage

```go
stdout, err := ffmpeg.StdoutPipe()
if err != nil {
	return errors.Wrap(err, "failed to get stdout pipe:", err)
	log.Fatalln("failed to get stdout pipe:", err)
}

if err := ffmpeg.Start(); err != nil {
	return errors.Wrap(err, "failed to start ffmpeg")
}

if err := oggreader.DecodeBuffered(voiceSession, stdout); err != nil {
	return errors.Wrap(err, "failed to decode ogg")
}

if err := ffmpeg.Wait(); err != nil {
	return errors.Wrap(err, "failed to finish ffmpeg")
}
```
