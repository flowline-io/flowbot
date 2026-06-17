package app

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/client"
)

const maxSplashSkills = 6
const splashNoSkills = "(no skills enabled)"

// minToolsTruncateWidth is the minimum terminal width for tools-line ellipsis slicing.
const minToolsTruncateWidth = 23

// minSkillTruncateWidth is the minimum terminal width for skill-line ellipsis slicing.
const minSkillTruncateWidth = 11

// RenderSplash builds the Hermes-style startup panel.
func RenderSplash(width int, info *client.ChatAgentInfo, sessionID, serverHost string, styles *Styles) string {
	if info == nil {
		return ""
	}
	title := fmt.Sprintf("Flowbot Agent %s · %s", displayVersion(info.Version), serverHost)
	toolsLine := strings.Join(toolNames(info.Tools), ", ")
	if width >= minToolsTruncateWidth && len(toolsLine) > width-20 {
		toolsLine = toolsLine[:width-23] + "..."
	}

	var skillLines []string
	for i, skill := range info.Skills {
		if i >= maxSplashSkills {
			skillLines = append(skillLines, fmt.Sprintf("(and %d more skills...)", info.SkillCount-maxSplashSkills))
			break
		}
		line := skill.Name
		if skill.Description != "" {
			line += ": " + skill.Description
		}
		if width >= minSkillTruncateWidth && len(line) > width-8 {
			line = line[:width-11] + "..."
		}
		skillLines = append(skillLines, line)
	}

	body := strings.Builder{}
	writeBuilder(&body, styles.BannerDim.Render(title))
	writeBuilder(&body, "\n\n")
	writeBuilder(&body, "Available Tools\n")
	writeBuilder(&body, toolsLine+"\n\n")
	writeBuilder(&body, "Available Skills\n")
	if len(skillLines) == 0 {
		writeBuilder(&body, splashNoSkills+"\n")
	} else {
		for _, line := range skillLines {
			writeBuilder(&body, line+"\n")
		}
	}
	writeBuilder(&body, "\n")
	writeBuilder(&body, fmt.Sprintf("%s · %s\n", info.ChatModel, info.Provider))
	writeBuilder(&body, fmt.Sprintf("Workspace: %s\n", info.Workspace))
	writeBuilder(&body, fmt.Sprintf("Session: %s\n\n", sessionID))
	writeBuilder(&body, fmt.Sprintf("%d tools · %d skills · /help", info.ToolCount, info.SkillCount))

	return styles.SplashBox.Width(width - 2).Render(body.String())
}

func toolNames(tools []client.ChatToolInfo) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}

// displayVersion normalizes server version strings to a single leading v prefix.
func displayVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "v")
	raw = strings.TrimPrefix(raw, "V")
	if raw == "" {
		return "dev"
	}
	return "v" + raw
}
