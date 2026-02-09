# High-Quality TUI Recording

Vibeauracle includes a sophisticated screen recording feature (`/record`) that captures TUI interactions and exports them as high-quality MP4 videos. Unlike traditional screen recorders that capture the entire OS desktop, this system records the internal terminal state, ensuring perfect clarity and minimal file size.

## Technical Architecture

The recording system is designed for high performance and visual quality, leveraging background processing and parallelized rendering.

### 1. Frame Capture (The Ticker)

When recording starts:
-   A high-frequency ticker (41ms interval, roughly 24 FPS) is started.
-   On every tick, the system checks if the TUI "view" has changed (`isDirty`).
-   If the view is unchanged, it simply increments a `ticks` counter on the last captured frame. This is a crucial optimization that avoids redundant rendering.
-   If the view has changed, it captures the new ANSI string as a new `recordedFrame`.

### 2. Parallel Rendering

When recording stops, the accumulated frames are processed in the background:
-   **Deduplication**: Since terminal frames often repeat, the system uses a thread-safe cache (Map + Mutex) to ensure each unique ANSI state is only rendered once.
-   **Multi-core Processing**: It utilizes a semaphore-based worker pool (matching `runtime.NumCPU()`) to render ANSI frames to raw RGB24 pixels in parallel using the same logic as the screenshot engine.
-   **Memory Efficiency**: Frames are rendered to raw bytes in memory before being piped to the encoder.

### 3. FFmpeg Pipe Integration

The final video is assembled using `ffmpeg`. Instead of saving temporary images to disk (which is slow and wears out SSDs), Vibeauracle pipes raw RGB24 data directly into `ffmpeg`'s stdin.

#### Command Configuration
The FFmpeg command is dynamically tuned based on the user's hardware:
-   **Encoders**: It checks for hardware acceleration:
    -   **macOS**: Uses `h264_videotoolbox` for Apple Silicon/Intel hardware encoding.
    -   **NVIDIA Systems**: Checks for `nvidia-smi` and `h264_nvenc` support in FFmpeg.
    -   **Fallback**: Defaults to `libx264` (CPU) for universal compatibility.
-   **Parameters**:
    -   `pix_fmt yuv420p`: Ensures compatibility with almost all video players.
    -   `vf scale=...`: Automatically scales the video to a maximum of 1920px width while maintaining aspect ratio and ensuring dimensions are even (required by most H.264 profiles).
    -   `movflags +faststart`: Optimizes the MP4 for web streaming.

## Why this is "Clever" Logic

1.  **Zero IO during capture**: Frames are stored as strings in RAM. No disk writes happen until the recording is finished, preventing lag during use.
2.  **Tick-based Duration**: By storing `ticks` per frame instead of actual timing, the system handles variable frame rates and terminal pauses perfectly.
3.  **Pipe Streaming**: Streaming raw pixels to FFmpeg is the fastest possible way to generate video, avoiding the overhead of file system metadata and encoding/decoding image formats like PNG.
4.  **Hardware Awareness**: The system detects the best encoder for the platform, ensuring the background processing is as fast as possible.

## Replicating the Logic

To replicate this:
1.  **State Management**: Create a `recordedFrame` struct that holds the string state and a duration/tick counter.
2.  **Efficient Rendering**: Use a library like `gg` to render ANSI to raw RGB bytes.
3.  **FFmpeg Integration**: Use `os/exec` to start an `ffmpeg` process with `-f rawvideo` and `-i -` (stdin).
4.  **Parallelize**: Use a worker pool to render unique frames. Most terminal interactions only involve a few unique frame states (e.g., typing), so caching is extremely effective.
