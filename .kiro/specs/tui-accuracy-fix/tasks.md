# Implementation Tasks: TUI Accuracy Fix

## Task 1: Enhance Progress Struct and Validation

- [x] 1.1 Add new fields to Progress struct (SpeedRaw, BitrateRaw, ETAAvailable, StartTime, LastValidFPS, LastValidSpeed, FrameEstimated)
- [x] 1.2 Implement clampPercentage function that ensures values are within 0-100 range
- [x] 1.3 Implement validate method on Progress struct to check all values for sanity
- [x] 1.4 Add previous value retention logic for invalid parsed values

## Task 2: Improve Frame Estimation

- [x] 2.1 Implement parseFrameRate function to handle fractional formats (e.g., "24000/1001")
- [x] 2.2 Enhance GetTotalFrames to set FrameEstimated flag appropriately
- [x] 2.3 Add dynamic TotalFrames adjustment when current frame exceeds estimate

## Task 3: Fix Progress Parsing

- [x] 3.1 Update bitrateRe regex to capture "N/A" and all FFmpeg formats
- [x] 3.2 Update speedRe regex to capture "N/A" and handle missing values
- [x] 3.3 Implement parseSpeed function returning (float64, string, bool)
- [x] 3.4 Implement parseBitrate function returning (string, string, bool)
- [x] 3.5 Add division-by-zero protection in all calculations

## Task 4: Improve ETA Calculation

- [x] 4.1 Implement calculateETA method with FPS-based and speed-based fallback
- [x] 4.2 Add warmup detection (first 2 seconds show "--:--")
- [x] 4.3 Implement ETA smoothing to prevent erratic jumps
- [x] 4.4 Return negative duration when ETA cannot be calculated

## Task 5: Enhance Display Formatting

- [x] 5.1 Implement formatSpeed function handling N/A and missing values
- [x] 5.2 Implement formatBitrate function handling N/A and missing values
- [x] 5.3 Implement formatETA function with warmup and unavailable handling
- [x] 5.4 Update renderEncodingView to use new formatting functions
- [x] 5.5 Show "Calculating..." for percentage when both frame count and duration unavailable

## Task 6: Completion Handling

- [x] 6.1 Implement GetActualOutputSize method to read file size from disk
- [x] 6.2 Update renderDoneView to use actual file size instead of progress data
- [x] 6.3 Handle file stat errors gracefully with fallback to progress TotalSize

## Task 7: Property-Based Tests

- [x] 7.1 Write property test for clampPercentage (Property 1)
- [x] 7.2 Write property test for frame-based percentage calculation (Property 2)
- [x] 7.3 Write property test for duration-based percentage calculation (Property 3)
- [x] 7.4 Write property test for frame estimation from duration (Property 4)
- [x] 7.5 Write property test for fractional frame rate parsing (Property 5)
- [x] 7.6 Write property test for FPS-based ETA calculation (Property 6)
- [x] 7.7 Write property test for speed-based ETA calculation (Property 7)
- [x] 7.8 Write property test for invalid speed returns unavailable ETA (Property 8)
- [x] 7.9 Write property test for speed parsing (Property 10)
- [x] 7.10 Write property test for bitrate parsing (Property 11)
- [x] 7.11 Write property test for file size formatting (Property 13)
- [x] 7.12 Write property test for division by zero protection (Property 16)
- [x] 7.13 Write property test for non-negative validation (Property 17)

## Task 8: Integration and Testing

- [x] 8.1 Run all property-based tests and fix any failures
- [ ] 8.2 Manual testing with real video files
- [ ] 8.3 Verify TUI displays accurate information throughout encoding lifecycle
