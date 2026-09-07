package parser

import (
	"testing"
)

func FuzzParseTaskWithCriteria(f *testing.F) {
	f.Add([]byte("---\ntitle: Seed Task\nstatus: backlog\n---\n# Test\n- [ ] test"), "fuzz-id")
	f.Add([]byte("garbage string without frontmatter"), "garbage-id")
	f.Add([]byte("---\nmalformed yaml\n---\nbody"), "malformed-id")

	f.Fuzz(func(t *testing.T, content []byte, id string) {
		ParseTaskWithCriteria(content, id)
		ParseStrategy(content, id)
		ParseMilestone(content, id)
		ParseGlossaryTerm(content, id)
	})
}
