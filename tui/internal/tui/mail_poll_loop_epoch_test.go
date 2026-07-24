package tui

import "testing"

func TestMailPollLoopEpochRejectsOldTickAfterSameGenerationReturn(t *testing.T) {
	app := App{currentView: appViewHelp}
	app.installMailModel(NewMailModel(
		t.TempDir(),
		"human",
		t.TempDir(),
		"",
		"",
		200,
		"",
		"en",
		false,
		0,
	))

	oldTick := tickMsg{generation: app.mail.generation}

	returnedModel, _ := app.Update(MarkdownViewerCloseMsg{})
	returned := returnedModel.(App)
	if returned.mail.generation != oldTick.generation {
		t.Fatalf("same-generation return changed Mail generation from %d to %d", oldTick.generation, returned.mail.generation)
	}
	serialAfterReturn := returned.mail.refreshRequestSerial

	updatedModel, cmd := returned.Update(oldTick)
	updated := updatedModel.(App)
	if updated.mail.refreshRequestSerial != serialAfterReturn {
		t.Fatalf(
			"old tick advanced refresh serial after same-generation return: got %d, want %d",
			updated.mail.refreshRequestSerial,
			serialAfterReturn,
		)
	}
	if cmd != nil {
		t.Fatal("old tick returned a refresh/rearm command after same-generation return")
	}
}
