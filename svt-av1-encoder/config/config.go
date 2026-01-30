package config

// Profile represents a named encoding profile
type Profile string

const (
	ProfileDefault  Profile = "default"  // Balanced quality/size (CRF 35)
	ProfileQuality  Profile = "quality"  // High quality, larger files (CRF 30)
	ProfilePodcast  Profile = "podcast"  // Optimized for talking heads (CRF 40)
	ProfileCompress Profile = "compress" // Maximum compression (CRF 45)
	ProfileFilm     Profile = "film"     // For movies/cinema (CRF 32, film grain)
)

// AvailableProfiles returns all available profile names
func AvailableProfiles() []Profile {
	return []Profile{ProfileDefault, ProfileQuality, ProfilePodcast, ProfileCompress, ProfileFilm}
}

// Config holds the encoder configuration settings
type Config struct {
	// Profile name for display purposes
	ProfileName Profile
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

// DefaultConfig returns the SVT-AV1-HDR standard defaults (balanced profile)
func DefaultConfig() Config {
	return GetProfile(ProfileDefault)
}

// GetProfile returns the configuration for a specific profile
func GetProfile(profile Profile) Config {
	// Base config with common settings
	base := Config{
		ProfileName:           profile,
		Tune:                  1, // PSNR
		VarianceBoost:         true,
		VarianceBoostStrength: 2,
		Sharpness:             1,
		TFStrength:            1,
		KFTFStrength:          1,
		ACBias:                1.0,
		SharpTX:               true,
		FilmGrain:             0,
		MaxSizePercent:        0,
		RemoveLanguages:       []string{},
		RemoveImageCodecs:     []string{"mjpeg", "png"},
		MinBitrate:            0,
	}

	switch profile {
	case ProfileQuality:
		// High quality - for important content you want to preserve well
		base.CRF = 30
		base.Preset = 3 // Slower for better quality
		base.VarianceBoostStrength = 3

	case ProfilePodcast:
		// Optimized for talking heads, podcasts, interviews, lectures
		// These have low motion and don't need high bitrate
		base.CRF = 40
		base.Preset = 5 // Faster since content is simple
		base.VarianceBoostStrength = 1

	case ProfileCompress:
		// Maximum compression - for archiving or storage constrained situations
		base.CRF = 45
		base.Preset = 6
		base.VarianceBoostStrength = 1
		base.Sharpness = 0

	case ProfileFilm:
		// For movies and cinematic content
		base.CRF = 32
		base.Preset = 2  // Slower for quality
		base.Tune = 0    // VQ tuning for visual quality
		base.FilmGrain = 8 // Preserve film grain
		base.VarianceBoostStrength = 3

	default: // ProfileDefault
		// Balanced quality/size - good for general content
		base.CRF = 35
		base.Preset = 4
	}

	return base
}

// ProfileDescription returns a human-readable description of a profile
func ProfileDescription(profile Profile) string {
	switch profile {
	case ProfileQuality:
		return "High quality (CRF 30) - For important content, larger files"
	case ProfilePodcast:
		return "Podcast/Talking heads (CRF 40) - Optimized compression for low-motion content"
	case ProfileCompress:
		return "Maximum compression (CRF 45) - Smallest files, some quality loss"
	case ProfileFilm:
		return "Film/Cinema (CRF 32) - Preserves film grain, high quality"
	default:
		return "Default balanced (CRF 35) - Good quality/size balance for general content"
	}
}
