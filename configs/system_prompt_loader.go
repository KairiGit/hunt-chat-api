package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SystemPromptConfig はsystem_prompt.yamlの構造を定義
type SystemPromptConfig struct {
	System struct {
		Role     string `yaml:"role"`
		Version  string `yaml:"version"`
		Language string `yaml:"language"`
	} `yaml:"system"`

	SystemInfo struct {
		Name     string `yaml:"name"`
		FullName string `yaml:"full_name"`
		Purpose  string `yaml:"purpose"`
		Features []struct {
			Name        string   `yaml:"name"`
			Description string   `yaml:"description"`
			Endpoint    string   `yaml:"endpoint,omitempty"`
			Endpoints   []string `yaml:"endpoints,omitempty"`
		} `yaml:"features"`
	} `yaml:"system_info"`

	TechStack struct {
		Frontend []struct {
			Name        string `yaml:"name"`
			Version     string `yaml:"version,omitempty"`
			Description string `yaml:"description"`
		} `yaml:"frontend"`
		Backend []struct {
			Name        string   `yaml:"name"`
			Version     string   `yaml:"version,omitempty"`
			Description string   `yaml:"description,omitempty"`
			Models      []string `yaml:"models,omitempty"`
		} `yaml:"backend"`
		Database []struct {
			Name        string   `yaml:"name"`
			Version     string   `yaml:"version,omitempty"`
			Description string   `yaml:"description"`
			Collections []string `yaml:"collections,omitempty"`
		} `yaml:"database"`
		Deployment []struct {
			Platform string `yaml:"platform"`
			Target   string `yaml:"target"`
		} `yaml:"deployment"`
	} `yaml:"tech_stack"`

	ResponseGuidelines []struct {
		Priority  int    `yaml:"priority"`
		Condition string `yaml:"condition"`
		Action    string `yaml:"action"`
	} `yaml:"response_guidelines"`

	Tone struct {
		Style         string `yaml:"style"`
		Personality   string `yaml:"personality"`
		LanguageLevel string `yaml:"language_level"`
	} `yaml:"tone"`

	Constraints []string `yaml:"constraints"`

	Examples struct {
		GoodResponses []struct {
			Question string `yaml:"question"`
			Answer   string `yaml:"answer"`
		} `yaml:"good_responses"`
		BadResponses []struct {
			Question   string `yaml:"question"`
			Answer     string `yaml:"answer,omitempty"`
			BadAnswer  string `yaml:"bad_answer,omitempty"`
			GoodAnswer string `yaml:"good_answer,omitempty"`
		} `yaml:"bad_responses"`
	} `yaml:"examples"`

	SpecialCommands struct {
		Help struct {
			Trigger  []string `yaml:"trigger"`
			Response string   `yaml:"response"`
		} `yaml:"help"`
		Docs struct {
			Trigger  []string `yaml:"trigger"`
			Response string   `yaml:"response"`
		} `yaml:"docs"`
	} `yaml:"special_commands"`

	Metadata struct {
		CreatedAt   string `yaml:"created_at"`
		LastUpdated string `yaml:"last_updated"`
		Version     string `yaml:"version"`
		Author      string `yaml:"author"`
	} `yaml:"metadata"`
}

var cachedSystemPrompt *SystemPromptConfig

// LoadSystemPrompt はYAMLファイルからシステムプロンプト設定を読み込む
func LoadSystemPrompt() (*SystemPromptConfig, error) {
	if cachedSystemPrompt != nil {
		return cachedSystemPrompt, nil
	}

	data, err := os.ReadFile("configs/system_prompt.yaml")
	if err != nil {
		return nil, fmt.Errorf("システムプロンプト設定ファイルの読み込みに失敗: %w", err)
	}

	var config SystemPromptConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("YAMLのパースに失敗: %w", err)
	}

	cachedSystemPrompt = &config
	return cachedSystemPrompt, nil
}

// BuildSystemPrompt は設定からシステムプロンプトを構築
func (c *SystemPromptConfig) BuildSystemPrompt() string {
	var sb strings.Builder

	// 役割の定義
	sb.WriteString(fmt.Sprintf("あなたは、%sです。\n\n", c.System.Role))

	// システム概要
	sb.WriteString("## システム概要\n")
	sb.WriteString(fmt.Sprintf("- **システム名**: %s (%s)\n", c.SystemInfo.Name, c.SystemInfo.FullName))
	sb.WriteString(fmt.Sprintf("- **目的**: %s\n", c.SystemInfo.Purpose))
	sb.WriteString("- **主要機能**:\n")
	for i, feature := range c.SystemInfo.Features {
		sb.WriteString(fmt.Sprintf("  %d. %s: %s\n", i+1, feature.Name, feature.Description))
		if feature.Endpoint != "" {
			sb.WriteString(fmt.Sprintf("     - API: %s\n", feature.Endpoint))
		}
		if len(feature.Endpoints) > 0 {
			sb.WriteString("     - API:\n")
			for _, ep := range feature.Endpoints {
				sb.WriteString(fmt.Sprintf("       * %s\n", ep))
			}
		}
	}
	sb.WriteString("\n")

	// 技術スタック
	sb.WriteString("## 技術スタック\n")

	// フロントエンド
	sb.WriteString("### フロントエンド\n")
	for _, tech := range c.TechStack.Frontend {
		if tech.Version != "" {
			sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", tech.Name, tech.Version, tech.Description))
		} else {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", tech.Name, tech.Description))
		}
	}
	sb.WriteString("\n")

	// バックエンド
	sb.WriteString("### バックエンド\n")
	for _, tech := range c.TechStack.Backend {
		if tech.Version != "" {
			sb.WriteString(fmt.Sprintf("- **%s** (%s)", tech.Name, tech.Version))
		} else {
			sb.WriteString(fmt.Sprintf("- **%s**", tech.Name))
		}
		if tech.Description != "" {
			sb.WriteString(fmt.Sprintf(": %s", tech.Description))
		}
		sb.WriteString("\n")
		if len(tech.Models) > 0 {
			sb.WriteString("  モデル:\n")
			for _, model := range tech.Models {
				sb.WriteString(fmt.Sprintf("  * %s\n", model))
			}
		}
	}
	sb.WriteString("\n")

	// データベース
	sb.WriteString("### データベース\n")
	for _, db := range c.TechStack.Database {
		if db.Version != "" {
			sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", db.Name, db.Version, db.Description))
		} else {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", db.Name, db.Description))
		}
		if len(db.Collections) > 0 {
			sb.WriteString("  コレクション:\n")
			for _, col := range db.Collections {
				sb.WriteString(fmt.Sprintf("  * %s\n", col))
			}
		}
	}
	sb.WriteString("\n")

	// デプロイ
	sb.WriteString("### デプロイ\n")
	for _, deploy := range c.TechStack.Deployment {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", deploy.Platform, deploy.Target))
	}
	sb.WriteString("\n")

	// 回答方針
	sb.WriteString("## 回答方針\n")
	for _, guideline := range c.ResponseGuidelines {
		sb.WriteString(fmt.Sprintf("%d. %s → %s\n", guideline.Priority, guideline.Condition, guideline.Action))
	}
	sb.WriteString("\n")

	// トーン
	sb.WriteString("## トーン\n")
	sb.WriteString(fmt.Sprintf("- スタイル: %s\n", c.Tone.Style))
	sb.WriteString(fmt.Sprintf("- パーソナリティ: %s\n", c.Tone.Personality))
	sb.WriteString(fmt.Sprintf("- 言語レベル: %s\n", c.Tone.LanguageLevel))
	sb.WriteString("\n")

	// 制約
	sb.WriteString("## 制約事項\n")
	for _, constraint := range c.Constraints {
		sb.WriteString(fmt.Sprintf("- %s\n", constraint))
	}
	sb.WriteString("\n")

	// 追加指示
	sb.WriteString("過去の会話履歴と現在の分析コンテキストを統合的に分析し、ユーザーの質問に的確に答えてください。\n")

	return sb.String()
}

// CheckSpecialCommand は特別なコマンドかチェック
func (c *SystemPromptConfig) CheckSpecialCommand(message string) (bool, string) {
	lowerMsg := strings.ToLower(message)

	// ヘルプコマンド
	for _, trigger := range c.SpecialCommands.Help.Trigger {
		if strings.Contains(lowerMsg, strings.ToLower(trigger)) {
			return true, c.SpecialCommands.Help.Response
		}
	}

	// ドキュメントコマンド
	for _, trigger := range c.SpecialCommands.Docs.Trigger {
		if strings.Contains(lowerMsg, strings.ToLower(trigger)) {
			return true, c.SpecialCommands.Docs.Response
		}
	}

	return false, ""
}
