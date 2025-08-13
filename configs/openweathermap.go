package config

// OpenWeatherMapConfig OpenWeatherMap API設定
type OpenWeatherMapConfig struct {
	APIKey  string
	BaseURL string
}

// GetOpenWeatherMapConfig OpenWeatherMap設定を取得
func GetOpenWeatherMapConfig() *OpenWeatherMapConfig {
	return &OpenWeatherMapConfig{
		APIKey:  getEnv("OPENWEATHERMAP_API_KEY", ""),
		BaseURL: getEnv("OPENWEATHERMAP_BASE_URL", "https://api.openweathermap.org/data/2.5"),
	}
}
