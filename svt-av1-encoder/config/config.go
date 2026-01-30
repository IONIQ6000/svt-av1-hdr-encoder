package config

// Config holds the encoder configuration settings
type Config struct {
	// CRF is the Constant Rate Factor (0-63, lower = better quality)
	// SVT-AV1-HDR default recommendation: start with 35
	CRF int
	// Preset controls encoding speed vs compression (0-13, lower = slower/better)
	// SVT-AV1-HDR recommends 2-6 for quality
	Preset int
	// Tune selects the tuning mode (0=VQ, 1=PSNR, 2=SSIM, 3=IQ, 4=Film Grain)
	// SVT-AV1-HDR default: 1 (PSNR)
	Tune int
	// VarianceBoost enables variance boost for better detail retention
	// SVT-AV1-HDR default: enabled
	VarianceBoost bool
	// VarianceBoostStrength controls variance boost intensity (1-4)
	// SVT-AV1-HDR default: 2
	VarianceBoostStrength int
	// Sharpness prioritizes encoder sharpness (0-2)
	// SVT-AV1-HDR default: 1
	Sharpness int
	// TFStrength controls temporal filtering strength for alt-ref frames
	// SVT-AV1-HDR default: 1 (much lower than mainline to reduce blur)
	TFStrength int
	// KFTFStrength controls temporal filtering strength for keyframes
	// SVT-AV1-HDR default: 1
	KFTFStrength int
	// ACBias strength of AC bias in rate distortion
	// SVT-AV1-HDR default: 1.0
	ACBias float64
	// SharpTX enables sharp transform optimizations
	// SVT-AV1-HDR default: enabled
	SharpTX bool
	// FilmGrain denoising level (0=off, 1-50=level)
	// SVT-AV1-HDR default: 0 (disabled, as it often harms visual fidelity)
	FilmGrain int
	// MaxSizePercent is the maximum output size as percentage of input (0 = disabled)
	MaxSizePercent int
	// RemoveLanguages is a list of language codes to remove from streams
	RemoveLanguages []string
	// RemoveImageCodecs is a list of image codecs to remove (e.g., mjpeg, png)
	RemoveImageCodecs []string
	// MinBitrate is the minimum source bitrate in kbps to allow encoding (0 = disabled)
	MinBitrate int
}

// DefaultConfig returns the SVT-AV1-HDR standard defaults
func DefaultConfig() Config {
	return Config{
		// SVT-AV1-HDR recommends starting with CRF 35 for general content
		CRF: 35,
		// Preset 4 is a good balance of speed and quality
		Preset: 4,
		// Tune 1 (PSNR) is the default
		Tune: 1,
		// Variance boost is enabled by default in SVT-AV1-HDR
		VarianceBoost:         true,
		VarianceBoostStrength: 2,
		// Sharpness 1 is default to prioritize encoder sharpness
		Sharpness: 1,
		// TF strength 1 for lower temporal filtering (less blur)
		TFStrength: 1,
		// KF TF strength 1 to remove keyframe artifacts
		KFTFStrength: 1,
		// AC bias 1.0 is default
		ACBias: 1.0,
		// Sharp TX enabled by default
		SharpTX: true,
		// Film grain denoising disabled by default (harms visual fidelity)
		FilmGrain: 0,
		// Size check disabled by default
		MaxSizePercent: 0,
		// Stream removal lists
		RemoveLanguages:   []string{},
		RemoveImageCodecs: []string{"mjpeg", "png"},
		MinBitrate:        0,
	}
}
