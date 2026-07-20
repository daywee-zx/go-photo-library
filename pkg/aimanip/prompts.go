package aimanip

const (
	PromptTagPhoto = `Analyze the provided image.

Return ONLY a valid JSON object. Do not use Markdown, code blocks, comments, or any additional text.

The JSON must have exactly the following structure:

{
  "description": "A concise factual description of the image in 1-3 sentences.",
  "tags": [
    "tag1",
    "tag2"
  ],
  "text": "All readable text found in the image. In English."
}

Requirements:
- The response must be valid JSON.
- Do not include any fields other than "description", "tags", and "text".
- The "description" should describe only what is clearly visible. Do not speculate about things that cannot be inferred from the image.
- The "tags" array should contain 5-20 short, lowercase keywords (one word) describing the main objects, people, actions, colors, locations, style, and topics. Do not include duplicate tags.
- The "text" field must contain every readable piece of text exactly as it appears, preserving line breaks whenever possible. Always translate the text to English. Do not summarize.
- If no text is visible, return an empty string for "text".
- If no suitable tags can be generated, return an empty array.
- The JSON object must start with '{' and end with '}'.`

	PromptTagRequest = `Analyze the provided image search request: "%s"

Return ONLY a valid JSON object. Do not use Markdown, code blocks, comments, or any additional text.

The JSON must have exactly the following structure:

{
  "tags": [
    "tag1",
    "tag2"
  ]
}

Requirements:
- The response must be valid JSON.
- Do not include any fields other than "tags".
- The "tags" array should contain 5-20 short, lowercase keywords (one word) describing the main objects, people, actions, colors, locations, style, and topics. Do not include duplicate tags.
- If no suitable tags can be generated, return an empty array.
- The JSON object must start with '{' and end with '}'.`
)
