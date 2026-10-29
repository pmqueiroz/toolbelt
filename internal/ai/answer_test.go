package ai

import (
	"reflect"
	"testing"
)

func TestParseReadsBlocks(t *testing.T) {
	raw := `{"blocks":[{"type":"command","text":"list files","code":"ls -la","language":"","items":[]}]}`
	got := Parse(raw)
	want := Answer{Blocks: []Block{{Type: Command, Text: "list files", Code: "ls -la"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse = %#v, want %#v", got, want)
	}
}

func TestParseStripsMarkdownFences(t *testing.T) {
	raw := "```json\n{\"blocks\":[{\"type\":\"text\",\"text\":\"Earth's natural satellite.\"}]}\n```"
	got := Parse(raw)
	if len(got.Blocks) != 1 || got.Blocks[0].Text != "Earth's natural satellite." {
		t.Fatalf("Parse = %#v", got)
	}
}

func TestParseFallsBackToTextWhenNotJSON(t *testing.T) {
	got := Parse("  just words  ")
	want := Answer{Blocks: []Block{{Type: Text, Text: "just words"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse = %#v, want %#v", got, want)
	}
}

func TestParseDropsUnknownAndEmptyBlocks(t *testing.T) {
	raw := `{"blocks":[{"type":"image","text":"x"},{"type":"command","code":" "},{"type":"list","items":["", " a "]},{"type":"warning","text":"careful"}]}`
	got := Parse(raw)
	want := Answer{Blocks: []Block{{Type: List, Items: []string{"a"}}, {Type: Warning, Text: "careful"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse = %#v, want %#v", got, want)
	}
}

func TestParseOfJSONWithoutValidBlocksShowsTheRawText(t *testing.T) {
	got := Parse(`{"answer":"42"}`)
	if len(got.Blocks) != 1 || got.Blocks[0].Type != Text || got.Blocks[0].Text != `{"answer":"42"}` {
		t.Fatalf("Parse = %#v", got)
	}
}

func TestParseOfEmptyAnswerHasNoBlocks(t *testing.T) {
	if got := Parse(" "); len(got.Blocks) != 0 {
		t.Fatalf("Parse = %#v", got)
	}
}

func TestCommandsListsOnlyCommandBlocks(t *testing.T) {
	a := Answer{Blocks: []Block{{Type: Text, Text: "x"}, {Type: Command, Code: "ls"}, {Type: Code, Code: "y"}, {Type: Command, Code: "pwd"}}}
	if got := a.Commands(); !reflect.DeepEqual(got, []string{"ls", "pwd"}) {
		t.Fatalf("Commands = %v", got)
	}
}

func TestSchemaRequiresEveryBlockField(t *testing.T) {
	block := Schema()["properties"].(map[string]any)["blocks"].(map[string]any)["items"].(map[string]any)
	required := block["required"].([]string)
	props := block["properties"].(map[string]any)
	if len(required) != len(props) {
		t.Fatalf("required %v does not cover properties %v", required, props)
	}
	if block["additionalProperties"] != false {
		t.Fatal("strict schemas need additionalProperties false")
	}
}
