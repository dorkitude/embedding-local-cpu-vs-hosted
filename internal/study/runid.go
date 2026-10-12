package study

import "os"

// RunID selects the study run used by reporting, settled querying and audits.
func RunID() string {
	if s := os.Getenv("STUDY_RUN_ID"); s != "" {
		return s
	}
	return "r20261012a"
}
