# TUI Screenshots

Vibeauracle features a high-fidelity screenshot system that captures the current terminal state and renders it into a beautiful, shareable image (PNG or SVG). This is not just a simple screen grab; it is a full reconstruction of the terminal's ANSI state into a polished graphical format.

## Overview

The screenshot feature is triggered by the `/shot` command within the TUI. It processes the raw ANSI output of the current view and applies a "modern terminal" aesthetic, complete with:

-   **Mac-style Window Controls**: Red, yellow, and green window dots.
-   **Soft Multi-layer Shadows**: Depth effects for a professional look.
-   **Rounded Corners**: A sleek, modern container.
-   **ANSI Color Fidelity**: Accurate mapping of 16-color and 256-color ANSI sequences to Hex colors.
-   **Font Consistency**: Uses `GoMono` (embedded) to ensure consistent rendering across platforms.

## Architectural Logic

The screenshot logic is split into two main parts: the capture mechanism and the rendering engine.

### 1. Capture Mechanism

When `/shot` is invoked, the TUI model:
1.  Retrieves the current view's string representation (including ANSI escape codes).
2.  Passes this string to the rendering engine.
3.  Determines a save path (configured in `config.UI.ScreenshotDir`).

### 2. Rendering Engine

The rendering engine (`cmd/vibeaura/shot_utils.go`) performs several sophisticated steps:

#### ANSI Sanitization
Standard TUI output contains many non-color escape sequences (e.g., cursor movements, alternate screen toggles, OSC sequences). The engine uses regex to strip everything except **SGR (Select Graphic Rendition)** sequences, which define colors and styles (bold, etc.).

#### Layout Calculation
The engine calculates the "visible" width and height:
-   It detects common border columns (e.g., vertical pipes `│`) to ensure content is nicely aligned.
-   It uses `runewidth` to account for wide Unicode characters (emojis, CJK).
-   It truncates lines that exceed the calculated max width to maintain a clean frame.

#### Canvas Rendering (Go Graphics)
Using the `gg` (Go Graphics) library, the engine:
1.  **Draws Shadows**: Multiple overlapping semi-transparent rectangles create a soft drop-shadow.
2.  **Draws the Frame**: A rounded rectangle with a dark background and a purple border (`#7D56F4`).
3.  **Draws Window Controls**: Three small colored circles in the top-left.
4.  **Renders Text**: Iterates through sanitized lines, parsing ANSI sequences on-the-fly to set the drawing color before placing text.

## Replicating the Logic

To replicate this in another tool:
1.  **Extract View**: Your TUI framework must be able to export its current state as a raw string with ANSI codes.
2.  **Font Embedding**: Embed a monospace TTF font (like GoMono) to ensure the output looks the same regardless of system fonts.
3.  **ANSI Parser**: Implement a robust state machine or regex parser for ANSI sequences. Supporting `1b[38;5;Nm` (256-color) and `1b[38;2;R;G;Bm` (TrueColor) is essential for modern terminal aesthetics.
4.  **Coordinate Mapping**: Map terminal "cells" to pixel coordinates based on font size and line height.
