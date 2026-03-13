package actions

import (
	"testing"
)

func TestTabList(t *testing.T) {
	m := newMockServer()
	m.response = `[{"id":"TAB1","url":"https://pinchtab.com"}]`
	defer m.close()
	client := m.server.Client()

	TabList(client, m.base(), "")
	if m.lastPath != "/tabs" {
		t.Errorf("expected /tabs, got %s", m.lastPath)
	}
}
