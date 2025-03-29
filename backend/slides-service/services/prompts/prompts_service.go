package prompts

import (
	"bytes"
	"text/template"

	"github.com/martin226/slideitin/backend/slides-service/models"
)

// Templates for different prompt types
const (
// Template for slide generation prompt
slideGenerationTemplate = `You are a domain expert with deep analytical capabilities and extensive experience in creating insightful Marp markdown presentations. You excel at critically analyzing documents, identifying key insights, drawing meaningful connections, and presenting complex information in a clear, impactful way. You have the ability to understand both explicit content and implicit implications within your domain of expertise.
	
Create a Marp markdown presentation using the following instructions:

The following is an example of how to create a Marp markdown presentation. All of the frontmatter in the example is also required for your response, other than the header and footer.

{{.ThemeExample}}

Theme: {{.Theme}}

{{.DetailLevel}}

{{.Audience}}

Generate the presentation content in Vietnamese.

IMPORTANT GUIDELINES:
1. Always begin with a short title slide with a title, a short description, and author name (only if provided). The title should be an H1 header, the description should be a regular text, and the author name should be a regular text.
2. Ensure that the content on each slide fits inside the slide. Never create paragraphs.
3. Always use bullet points and other formatting options to make the content more readable. 
4. Prefer multi-line code blocks over inline code blocks for any code longer than a few words. Even if the code is a single line, use a multi-line code block.
5. Do not end with --- (three dashes) on a new line, since this will end the presentation with an empty slide.

Make the slides look as beautiful and well-designed as possible. Use all of the formatting options available to you.

Enclose your response in triple backticks like this:

` + "```md" + `
<your response here>
` + "```"

	// Common markdown header template used across all themes
	commonMarpHeader = `---
marp: true
theme: {{.Theme}}
{{if .UseLeadClass -}}
_class: lead
{{- end}}
paginate: true
header: This is an optional header {{.HeaderLocation}}
footer: This is an optional footer {{.FooterLocation}}
---
{{if .HasTitleClass}}
<!-- _class: title -->
{{end}}
# Title

`

	// Common markdown body template for examples
	commonExampleBody = `## Heading 2

- {{.ThemeDescription}}
{{ if .HasInvertClass}}

---

<!-- _class: invert -->

## Inverted color scheme

- You can use the <!-- _class: invert --> tag at the top of a slide to create a dark mode slide.
- Use this when you want to have a slide with a different color scheme than the rest of the presentation.
- Do this when a slide should stand out.
{{end}}{{if .HasTinyTextClass}}

---

<!-- _class: tinytext -->

# Tinytext class

- You can use the <!-- _class: tinytext --> tag at the top of a slide to make some text tiny.
- This might be useful for References.
{{end}}

---

## Code blocks

### Multi-line code blocks

` + "```" + `python
print("This is a code block")
print("You can use triple backticks to create a code block")
print("You can also use the language name to highlight the code block")
` + "```" + `

- **Another example:**

` + "```" + `c
printf("This is another code block");
printf("Always specify the language name for code blocks");
` + "```" + `

---

### Inline code blocks

- ` + "`" + `this` + "`" + ` is an inline code block
- You can use it using single backticks like this: ` + "`" + `this` + "`" + `

---

## Creating new slides

- To create a new slide, use a new line with three dashes like this:

` + "```" + `
---

# New slide
` + "```" + `

---

# Conclusion

- You can use Markdown formatting to create **bold**, *italic*, and ~~strikethrough~~ text.
> This is a block quote
This is regular text`
)

// Theme configurations
var themeConfigs = map[string]map[string]interface{}{
	"default": {
		"UseLeadClass":    true,
		"HasInvertClass":  true,
		"HasTinyTextClass": false,
		"HasTitleClass":   false,
		"HeaderLocation":  "(top left of the slide)",
		"FooterLocation":  "(bottom left of the slide)",
		"ThemeDescription": "By default, the color scheme for each slide is light.",
	},
	"beam": {
		"UseLeadClass":    false,
		"HasInvertClass":  false,
		"HasTinyTextClass": true,
		"HasTitleClass":   true,
		"HeaderLocation":  "(bottom left half of the slide)",
		"FooterLocation":  "(bottom right half of the slide)",
		"ThemeDescription": "IMPORTANT: You must use the above title class tag at the top of the title slide (<!-- _class: title -->).\n- Beam is a light color scheme based on the LaTeX Beamer theme.",
	},
	"rose-pine": {
		"UseLeadClass":    true,
		"HasInvertClass":  false,
		"HasTinyTextClass": false,
		"HasTitleClass":   false,
		"HeaderLocation":  "(top left of the slide)",
		"FooterLocation":  "(bottom left of the slide)",
		"ThemeDescription": "Rose Pine is a dark color scheme.",
	},
	"gaia": {
		"UseLeadClass":    true,
		"HasInvertClass":  true,
		"HasTinyTextClass": false,
		"HasTitleClass":   false,
		"HeaderLocation":  "(top left of the slide)",
		"FooterLocation":  "(bottom left of the slide)",
		"ThemeDescription": "By default, the color scheme for each slide is light.",
	},
	"uncover": {
		"UseLeadClass":    true,
		"HasInvertClass":  true,
		"HasTinyTextClass": false,
		"HasTitleClass":   false,
		"HeaderLocation":  "(top middle of the slide)",
		"FooterLocation":  "(bottom middle of the slide)",
		"ThemeDescription": "By default, the color scheme for each slide is light.",
	},
	"graph_paper": {
		"UseLeadClass":    true,
		"HasInvertClass":  false,
		"HasTinyTextClass": true,
		"HasTitleClass":   false,
		"HeaderLocation":  "(top left of the slide)",
		"FooterLocation":  "(bottom left of the slide)",
		"ThemeDescription": "Graph Paper is a light color scheme.",
	},
}

// GenerateSlidePrompt creates a prompt for slide generation based on the given parameters
func GenerateSlidePrompt(theme string, settings models.SlideSettings) (string, error) {
	// Generate theme example
	themeExample, err := generateThemeExample(theme)
	if err != nil {
		return "", err
	}

	detailPrompt := ""
	if settings.SlideDetail == "detailed" {
		detailPrompt = "Perform a comprehensive analysis of the document, identifying both explicit content and deeper insights. Include all major sections and subsections, but go beyond simple extraction to draw meaningful connections and highlight important implications. For each topic, present both the key points and your expert analysis of their significance, potential impacts, and relationships to other concepts. Include relevant context that helps understand the broader implications. Identify patterns, trends, and potential future implications where relevant. Structure the content to build a cohesive narrative while maintaining visual clarity with 6-8 bullet points per slide. Each slide should not just present information, but contribute to a deeper understanding of the topic."
	} else if settings.SlideDetail == "medium" {
		detailPrompt = "Analyze and synthesize the key information from each section, identifying the most important concepts and their implications. While being selective with content, ensure you draw meaningful connections and highlight significant insights that support the core message. Look for patterns and relationships between different sections that reveal deeper understanding. Present your expert analysis alongside the main points, offering perspective on their significance and potential applications. Create a balanced narrative that combines factual content with insightful analysis. Limit each slide to 4-6 bullet points to maintain clarity while ensuring each point delivers valuable understanding."
	} else if settings.SlideDetail == "minimal" {
		detailPrompt = "Identify and analyze the most crucial elements of the document, focusing on key conclusions and their strategic importance. While being highly selective with content, ensure each point chosen represents a significant insight or critical understanding. Look for overarching themes and relationships that provide deeper meaning to the individual points. Transform raw conclusions into actionable insights that demonstrate expert-level understanding. Each slide should deliver high-impact content that combines essential information with valuable analysis. Maintain conciseness with 3-4 bullet points per slide while ensuring each point delivers meaningful value and perspective."
	}

	audiencePrompt := ""
	if settings.Audience == "general" {
		audiencePrompt = "Format the presentation for a general audience while maintaining depth of insight. Transform complex concepts into clear, accessible explanations that reveal their true significance. When technical terms appear, provide concise explanations that help build understanding. Look for real-world implications and practical relevance in the content. Create a narrative that not only explains what things are, but why they matter and how they connect to broader themes. Identify insights that would be valuable to someone encountering this topic for the first time. Structure the presentation to build understanding progressively while maintaining engagement through relevant examples and clear implications."
	} else if settings.Audience == "academic" {
		audiencePrompt = "Format the presentation for an academic audience with an emphasis on scholarly insight and theoretical depth. Analyze the content through relevant theoretical frameworks, identifying connections to broader academic discourse. Maintain methodological rigor while highlighting novel contributions and potential research implications. Look for gaps in current understanding that the content might address. Draw connections between different theoretical perspectives or methodological approaches present in the material. Identify potential areas for future research or theoretical development. When presenting findings, emphasize both their empirical significance and theoretical implications. Structure the argument to contribute to academic discourse while maintaining scholarly precision."
	} else if settings.Audience == "technical" {
		audiencePrompt = "Format the presentation for a technical audience with deep domain expertise. Analyze technical content to highlight not just specifications and implementations, but also architectural decisions, trade-offs, and technical implications. Identify potential technical challenges, optimization opportunities, and system-level impacts. When presenting technical solutions, include analysis of scalability, maintainability, and potential future considerations. Draw connections between different technical components to reveal system-level insights. For code or technical diagrams, provide expert-level analysis of design patterns, architectural principles, and best practices. Structure the content to build a comprehensive technical understanding while highlighting critical decision points and their implications."
	} else if settings.Audience == "professional" {
		audiencePrompt = "Format the presentation for business professionals with a focus on strategic insights and market implications. Analyze the content to identify business opportunities, competitive advantages, and potential market impacts. Transform data and findings into actionable business insights that inform decision-making. When presenting case studies or results, emphasize lessons learned and their broader applicability. Identify industry trends, market dynamics, and business model implications within the content. Look for insights about customer needs, market gaps, or operational improvements. For metrics and data, provide analysis of their business significance and strategic implications. Structure the presentation to build a compelling business case while highlighting key decision points and opportunities."
	} else if settings.Audience == "executive" {
		audiencePrompt = "Format the presentation for executive decision-makers with emphasis on strategic vision and organizational impact. Analyze the content to identify major strategic opportunities, risks, and competitive implications. Transform detailed findings into high-level insights that inform executive decision-making. When presenting outcomes or metrics, emphasize their impact on organizational strategy and market position. Look for insights about industry disruption, market transformation, or emerging opportunities. Identify implications for organizational capabilities, resource allocation, and competitive positioning. For financial or operational data, provide strategic context and long-term implications. Structure the presentation around key strategic decisions while maintaining focus on sustainable competitive advantage and organizational growth."
	}

	// Create template data
	data := map[string]interface{}{
		"Theme":        theme,
		"ThemeExample": themeExample,
		"DetailLevel":  detailPrompt,
		"Audience":     audiencePrompt,
	}

	// Parse and execute the template
	tmpl, err := template.New("slidePrompt").Parse(slideGenerationTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateThemeExample generates an example for a specific theme
func generateThemeExample(theme string) (string, error) {
	// Get theme configuration or use default config if theme doesn't exist
	themeConfig, exists := themeConfigs[theme]
	if !exists {
		themeConfig = themeConfigs["default"]
	}
	
	// Copy the theme config and add the theme name
	templateData := make(map[string]interface{})
	for k, v := range themeConfig {
		templateData[k] = v
	}
	templateData["Theme"] = theme

	// Generate the header
	headerTemplate, err := template.New("header").Parse(commonMarpHeader)
	if err != nil {
		return "", err
	}
	
	var headerBuf bytes.Buffer
	if err := headerTemplate.Execute(&headerBuf, templateData); err != nil {
		return "", err
	}
	
	// Generate the body
	bodyTemplate, err := template.New("body").Parse(commonExampleBody)
	if err != nil {
		return "", err
	}
	
	var bodyBuf bytes.Buffer
	if err := bodyTemplate.Execute(&bodyBuf, templateData); err != nil {
		return "", err
	}
	
	// Combine the parts into a complete example
	example := "```md\n" + headerBuf.String() + bodyBuf.String() + "\n```"
	
	return example, nil
}

// GenerateCustomPrompt creates a prompt from a custom template and parameters
func GenerateCustomPrompt(promptTemplate string, params map[string]interface{}) (string, error) {
	tmpl, err := template.New("customPrompt").Parse(promptTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", err
	}

	return buf.String(), nil
}
