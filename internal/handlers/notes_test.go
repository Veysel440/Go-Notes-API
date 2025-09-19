package handlers

import (
	"testing"
	"time"

	"github.com/Veysel440/go-notes-api/internal/repos"
)

func Test_noteETag_ChangesWithUpdate(t *testing.T) {
	n1 := repos.Note{ID: 10, Title: "t", Body: "b", CreatedAt: time.Unix(1000, 0)}
	n1.UpdatedAt = n1.CreatedAt
	e1 := noteETag(n1)

	n1.Body = "b2"
	n1.UpdatedAt = n1.CreatedAt.Add(time.Second)
	e2 := noteETag(n1)

	if e1 == e2 {
		t.Fatalf("etag should change on update; got %s", e1)
	}
}

func Test_collETag_DiffersWithParams(t *testing.T) {
	items := []repos.Note{
		{ID: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 5, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	e1 := collETag(1, 20, "a", "id", items)
	e2 := collETag(2, 20, "a", "id", items)
	if e1 == e2 {
		t.Fatalf("etag should differ for different page; %s", e1)
	}
}
