package bridge

import "testing"

func TestApplyNodeDOMMetadataRedactsPassword(t *testing.T) {
	node := A11yNode{
		Role:  "textbox",
		Name:  "Password",
		Value: "supersecret123",
	}
	meta := nodeDOMMetadata{
		Tag:       "input",
		InputType: "password",
	}
	applyNodeDOMMetadata(&node, meta)

	if node.Value != "••••••••" {
		t.Errorf("password value = %q, want redacted", node.Value)
	}
	if node.Tag != "input" {
		t.Errorf("tag = %q, want input", node.Tag)
	}
}

func TestApplyNodeDOMMetadataPreservesTextInput(t *testing.T) {
	node := A11yNode{
		Role:  "textbox",
		Name:  "Username",
		Value: "mario",
	}
	meta := nodeDOMMetadata{
		Tag:       "input",
		InputType: "text",
	}
	applyNodeDOMMetadata(&node, meta)

	if node.Value != "mario" {
		t.Errorf("text value = %q, want mario", node.Value)
	}
}

func TestApplyNodeDOMMetadataRedactsPasswordCaseInsensitive(t *testing.T) {
	node := A11yNode{
		Role:  "textbox",
		Name:  "Password",
		Value: "secret",
	}
	meta := nodeDOMMetadata{
		Tag:       "input",
		InputType: "Password",
	}
	applyNodeDOMMetadata(&node, meta)

	if node.Value != "••••••••" {
		t.Errorf("password value = %q, want redacted", node.Value)
	}
}

func TestApplyNodeDOMMetadataNoInputType(t *testing.T) {
	node := A11yNode{
		Role:  "textbox",
		Name:  "Search",
		Value: "query",
	}
	meta := nodeDOMMetadata{
		Tag: "input",
	}
	applyNodeDOMMetadata(&node, meta)

	if node.Value != "query" {
		t.Errorf("value = %q, want query", node.Value)
	}
}

func TestApplyNodeDOMMetadataEmptyPasswordValue(t *testing.T) {
	node := A11yNode{
		Role: "textbox",
		Name: "Password",
	}
	meta := nodeDOMMetadata{
		Tag:       "input",
		InputType: "password",
	}
	applyNodeDOMMetadata(&node, meta)

	if node.Value != "••••••••" {
		t.Errorf("empty password = %q, want redacted placeholder", node.Value)
	}
}
