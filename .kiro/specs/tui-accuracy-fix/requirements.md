# Requirements Document

## Introduction

This document specifies requirements for fixing TUI display accuracy issues in the svt-av1-encoder Go application. The TUI currently displays inaccurate progress information including percentage, ETA, speed, bitrate, and file size values. These fixes will ensure users see reliable, validated information throughout the encoding process.

## Glossary

- **TUI**: Terminal User Interface - the Bubble Tea-based display component
- **Progress_Parser**: The component that extracts progress data from FFmpeg output
- **Progress_Display**: The component that renders progress data to the terminal
- **Frame_Estimator**: The component that determines total frame count for percentage calculation
- **ETA_Calculator**: The component that computes estimated time remaining
- **FFmpeg**: The external video encoding tool that outputs progress data

## Requirements

### Requirement 1: Progress Percentage Accuracy

**User Story:** As a user, I want the progress percentage to always show accurate values between 0-100%, so that I can reliably track encoding progress.

#### Acceptance Criteria

1. THE Progress_Display SHALL clamp percentage values to the range 0-100 before display
2. WHEN frame count is available, THE Progress_Parser SHALL calculate percentage as (current_frame / total_frames) * 100
3. WHEN frame count is unavailable but duration is available, THE Progress_Parser SHALL calculate percentage from time progress
4. WHEN both frame count and duration are unavailable, THE Progress_Display SHALL show "Calculating..." instead of 0%
5. IF percentage calculation produces a value exceeding 100, THEN THE Progress_Parser SHALL cap it at 100

### Requirement 2: Frame Count Estimation

**User Story:** As a user, I want accurate frame count estimation, so that progress percentage is reliable.

#### Acceptance Criteria

1. WHEN probing for frame count, THE Frame_Estimator SHALL first attempt to read nb_frames from container metadata
2. WHEN container metadata is unavailable, THE Frame_Estimator SHALL calculate frames from duration and frame rate
3. WHEN frame rate is expressed as a fraction (e.g., "24000/1001"), THE Frame_Estimator SHALL correctly parse and calculate it
4. WHEN frame estimation fails completely, THE Frame_Estimator SHALL set TotalFrames to 0 and log a warning
5. THE Frame_Estimator SHALL handle variable frame rate content by using average frame rate

### Requirement 3: ETA Calculation Accuracy

**User Story:** As a user, I want accurate ETA estimates, so that I can plan around encoding completion time.

#### Acceptance Criteria

1. WHEN FPS is greater than 0 and total frames are known, THE ETA_Calculator SHALL compute ETA as remaining_frames / current_fps
2. WHEN FPS is 0 or unavailable, THE ETA_Calculator SHALL use speed-based calculation if speed is available
3. WHEN speed value is "N/A" or unparseable, THE ETA_Calculator SHALL return a negative duration to indicate unavailable
4. WHEN encoding has just started (less than 2 seconds), THE Progress_Display SHALL show "--:--" for ETA
5. THE ETA_Calculator SHALL smooth ETA values to prevent erratic jumps between updates

### Requirement 4: Speed Display Handling

**User Story:** As a user, I want the speed display to handle all FFmpeg output formats, so that I always see meaningful speed information.

#### Acceptance Criteria

1. WHEN FFmpeg outputs a numeric speed like "1.5x", THE Progress_Parser SHALL capture and display it
2. WHEN FFmpeg outputs "N/A" for speed, THE Progress_Display SHALL show "N/A" instead of empty string
3. WHEN FFmpeg outputs no speed value, THE Progress_Display SHALL show "..." until speed becomes available
4. THE Progress_Parser SHALL handle speed values with varying decimal precision (e.g., "1x", "1.5x", "1.23x")

### Requirement 5: Bitrate Display Handling

**User Story:** As a user, I want accurate bitrate display, so that I can monitor encoding quality.

#### Acceptance Criteria

1. THE Progress_Parser SHALL capture bitrate values in all FFmpeg formats (e.g., "1234kbits/s", "1.2Mbits/s", "N/A")
2. WHEN bitrate is "N/A" or unavailable, THE Progress_Display SHALL show "N/A" instead of empty string
3. WHEN bitrate value changes, THE Progress_Display SHALL update immediately
4. THE Progress_Parser SHALL normalize bitrate display to a consistent format

### Requirement 6: File Size Display Accuracy

**User Story:** As a user, I want accurate file size display during and after encoding, so that I can monitor output size.

#### Acceptance Criteria

1. WHEN FFmpeg reports total_size, THE Progress_Parser SHALL update TotalSize immediately
2. WHEN encoding completes, THE Progress_Display SHALL read actual file size from disk instead of using progress data
3. WHEN total_size is 0 or unavailable early in encoding, THE Progress_Display SHALL show "..." until available
4. THE Progress_Display SHALL format file sizes consistently using binary units (KiB, MiB, GiB)

### Requirement 7: Edge Case Handling

**User Story:** As a user, I want the TUI to handle edge cases gracefully, so that I never see broken or confusing displays.

#### Acceptance Criteria

1. WHEN FPS is 0, THE Progress_Display SHALL show "0.0" for FPS and handle ETA calculation gracefully
2. WHEN any progress value is missing or invalid, THE Progress_Display SHALL show a placeholder instead of empty or zero
3. WHEN frame count exceeds estimated total, THE Progress_Parser SHALL update TotalFrames to current frame count
4. WHEN encoding duration is very short (under 1 second), THE Progress_Display SHALL handle rapid completion gracefully
5. IF division by zero would occur in any calculation, THEN THE Progress_Parser SHALL return a safe default value

### Requirement 8: Progress Data Validation

**User Story:** As a user, I want all displayed values to be validated, so that I never see nonsensical data.

#### Acceptance Criteria

1. THE Progress_Parser SHALL validate that frame count is non-negative before use
2. THE Progress_Parser SHALL validate that FPS is non-negative before use
3. THE Progress_Parser SHALL validate that file size is non-negative before use
4. WHEN a parsed value fails validation, THE Progress_Parser SHALL retain the previous valid value
5. THE Progress_Display SHALL validate all values before rendering to prevent display corruption
