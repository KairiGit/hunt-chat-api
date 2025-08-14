package services

import (
	"testing"
)

func TestNewWeatherService(t *testing.T) {
	service := NewWeatherService()

	if service == nil {
		t.Fatal("NewWeatherService() returned nil")
	}

	if service.client == nil {
		t.Fatal("WeatherService client is nil")
	}
}

func TestGetRegionCodes(t *testing.T) {
	service := NewWeatherService()
	regionCodes := service.GetRegionCodes()

	if len(regionCodes) == 0 {
		t.Fatal("GetRegionCodes() returned empty map")
	}

	// 東京都のコードが存在することを確認
	if _, exists := regionCodes["130000"]; !exists {
		t.Error("Tokyo region code (130000) not found in region codes")
	}

	// 三重県のコードが存在することを確認
	if _, exists := regionCodes["240000"]; !exists {
		t.Error("Mie region code (240000) not found in region codes")
	}
}

func TestGetRegionName(t *testing.T) {
	service := NewWeatherService()

	testCases := []struct {
		regionCode string
		expected   string
	}{
		{"130000", "東京都"},
		{"240000", "三重県"},
		{"999999", "不明な地域"},
	}

	for _, tc := range testCases {
		result := service.getRegionName(tc.regionCode)
		if result != tc.expected {
			t.Errorf("getRegionName(%s) = %s, expected %s",
				tc.regionCode, result, tc.expected)
		}
	}
}

func TestGetAvailableHistoricalDataRange(t *testing.T) {
	service := NewWeatherService()
	dataRange := service.GetAvailableHistoricalDataRange()

	if len(dataRange) == 0 {
		t.Fatal("GetAvailableHistoricalDataRange() returned empty map")
	}

	// 期待されるキーが存在することを確認
	expectedKeys := []string{"start_date", "end_date", "max_range_days", "data_sources"}
	for _, key := range expectedKeys {
		if _, exists := dataRange[key]; !exists {
			t.Errorf("Expected key '%s' not found in data range", key)
		}
	}
}
