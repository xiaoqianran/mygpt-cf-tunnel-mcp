package agent

import "testing"

func TestCallRequestToolArguments(t *testing.T) {
	t.Run("arguments json", func(t *testing.T) {
		r := callRequest{ArgumentsJSON: `{"query":"manager call path","maxFiles":8}`}
		args, err := r.toolArguments()
		if err != nil {
			t.Fatal(err)
		}
		if args["query"] != "manager call path" || args["maxFiles"] != float64(8) {
			t.Fatalf("unexpected arguments: %#v", args)
		}
	})

	t.Run("legacy object", func(t *testing.T) {
		r := callRequest{Arguments: map[string]any{"query": "legacy"}}
		args, err := r.toolArguments()
		if err != nil || args["query"] != "legacy" {
			t.Fatalf("args=%#v err=%v", args, err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := (callRequest{ArgumentsJSON: `[]`}).toolArguments()
		if err == nil {
			t.Fatal("expected JSON object error")
		}
	})
}
