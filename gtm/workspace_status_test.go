package gtm

import (
	"testing"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Each type and ID must match a collection key and its ID field in a container
// version, so a change can be paired with the same entity in `versions get`.
func TestDescribeEntityMatchesVersionCollections(t *testing.T) {
	for _, tc := range []struct {
		entity *tagmanager.Entity
		want   EntityRef
	}{
		{&tagmanager.Entity{Tag: &tagmanager.Tag{TagId: "1", Name: "T"}}, EntityRef{"tag", "1", "T"}},
		{&tagmanager.Entity{Trigger: &tagmanager.Trigger{TriggerId: "2", Name: "R"}}, EntityRef{"trigger", "2", "R"}},
		{&tagmanager.Entity{Variable: &tagmanager.Variable{VariableId: "3", Name: "V"}}, EntityRef{"variable", "3", "V"}},
		{&tagmanager.Entity{Folder: &tagmanager.Folder{FolderId: "4", Name: "F"}}, EntityRef{"folder", "4", "F"}},
		{&tagmanager.Entity{BuiltInVariable: &tagmanager.BuiltInVariable{Type: "pageUrl", Name: "Page URL"}}, EntityRef{"builtInVariable", "pageUrl", "Page URL"}},
		{&tagmanager.Entity{CustomTemplate: &tagmanager.CustomTemplate{TemplateId: "5", Name: "C"}}, EntityRef{"customTemplate", "5", "C"}},
		{&tagmanager.Entity{Client: &tagmanager.Client{ClientId: "6", Name: "GA4"}}, EntityRef{"client", "6", "GA4"}},
		{&tagmanager.Entity{Transformation: &tagmanager.Transformation{TransformationId: "7", Name: "X"}}, EntityRef{"transformation", "7", "X"}},
		{&tagmanager.Entity{Zone: &tagmanager.Zone{ZoneId: "8", Name: "Z"}}, EntityRef{"zone", "8", "Z"}},
		{&tagmanager.Entity{GtagConfig: &tagmanager.GtagConfig{GtagConfigId: "9", Type: "googtag"}}, EntityRef{"gtagConfig", "9", ""}},
		{&tagmanager.Entity{}, EntityRef{}},
		{nil, EntityRef{}},
	} {
		got, value := describeEntity(tc.entity)
		if got != tc.want {
			t.Errorf("describeEntity(%+v) = %+v, want %+v", tc.entity, got, tc.want)
		}
		if (value == nil) != (tc.want.Type == "") {
			t.Errorf("describeEntity(%+v) returned entity %v", tc.entity, value)
		}
	}
}
