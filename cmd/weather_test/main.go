package main

import (
	"fmt"
	"hunt-chat-api/internal/services"
	"log"
)

func main() {
	fmt.Println("=== 気象庁API 詳細テスト ===")
	
	weatherService := services.NewWeatherService()
	
	// 複数地域のテスト
	regions := []string{
		"130000", // 東京都
		"240000", // 三重県（鈴鹿市）
		"270000", // 大阪府
	}
	
	for _, regionCode := range regions {
		fmt.Printf("\n--- 地域コード: %s ---\n", regionCode)
		
		forecastData, err := weatherService.GetForecastData(regionCode)
		if err != nil {
			log.Printf("エラー: %v", err)
			continue
		}
		
		fmt.Printf("取得データ数: %d件\n", len(forecastData))
		
		if len(forecastData) > 0 {
			fmt.Printf("発表機関: %s\n", forecastData[0].PublishingOffice)
			fmt.Printf("発表日時: %s\n", forecastData[0].ReportDatetime)
			
			// 各地域の詳細情報を表示
			for i, timeSeries := range forecastData[0].TimeSeries {
				fmt.Printf("\n  タイムシリーズ %d:\n", i+1)
				fmt.Printf("  時刻定義数: %d\n", len(timeSeries.TimeDefines))
				
				for j, area := range timeSeries.Areas {
					fmt.Printf("    地域 %d: %s (%s)\n", j+1, area.Area.Name, area.Area.Code)
					
					if len(area.Weathers) > 0 {
						fmt.Printf("      天気: %s\n", area.Weathers[0])
					}
					if len(area.WeatherCodes) > 0 {
						fmt.Printf("      天気コード: %s\n", area.WeatherCodes[0])
					}
					if len(area.Winds) > 0 {
						fmt.Printf("      風: %s\n", area.Winds[0])
					}
					if len(area.Pops) > 0 {
						fmt.Printf("      降水確率: %s%%\n", area.Pops[0])
					}
					if len(area.Temps) > 0 {
						fmt.Printf("      気温: %s℃\n", area.Temps[0])
					}
				}
			}
		}
	}
	
	fmt.Println("\n=== テスト完了 ===")
}
