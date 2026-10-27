package report

import (
	"testing"

	openreportsv1alpha1 "github.com/openreports/reports-api/apis/openreports.io/v1alpha1"
)

func TestAddResult(t *testing.T) {
	t.Parallel()
	t.Run("adds new result when none exist", func(t *testing.T) {
		t.Parallel()
		pol := &openreportsv1alpha1.Report{}
		result := openreportsv1alpha1.ReportResult{
			Source: "test-source",
			Policy: "test-policy",
			Rule:   "test-rule",
			Result: openreportsv1alpha1.Result("fail"),
		}

		addResult(pol, result)

		if len(pol.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(pol.Results))
		}
		if pol.Results[0].Source != "test-source" {
			t.Errorf("expected source 'test-source', got %q", pol.Results[0].Source)
		}
		if pol.Summary.Fail != 1 {
			t.Errorf("expected summary fail count 1, got %d", pol.Summary.Fail)
		}
	})

	t.Run("updates existing result with matching source, policy, and rule", func(t *testing.T) {
		t.Parallel()
		pol := &openreportsv1alpha1.Report{
			Results: []openreportsv1alpha1.ReportResult{
				{
					Source: "test-source",
					Policy: "test-policy",
					Rule:   "test-rule",
					Result: openreportsv1alpha1.Result("fail"),
					Properties: map[string]string{
						propCount: "5",
					},
				},
			},
		}
		newResult := openreportsv1alpha1.ReportResult{
			Source: "test-source",
			Policy: "test-policy",
			Rule:   "test-rule",
			Result: openreportsv1alpha1.Result("fail"),
			Properties: map[string]string{
				"protocol": "TCP",
			},
		}

		addResult(pol, newResult)

		if len(pol.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(pol.Results))
		}
		if pol.Results[0].Properties[propCount] != "6" {
			t.Errorf("expected count 6, got %q", pol.Results[0].Properties[propCount])
		}
		if pol.Results[0].Properties["protocol"] != "TCP" {
			t.Errorf("expected protocol 'TCP', got %q", pol.Results[0].Properties["protocol"])
		}
	})

	t.Run("adds new result when source differs", func(t *testing.T) {
		t.Parallel()
		pol := &openreportsv1alpha1.Report{
			Results: []openreportsv1alpha1.ReportResult{
				{
					Source: "source-1",
					Policy: "test-policy",
					Rule:   "test-rule",
				},
			},
		}
		newResult := openreportsv1alpha1.ReportResult{
			Source: "source-2",
			Policy: "test-policy",
			Rule:   "test-rule",
		}

		addResult(pol, newResult)

		if len(pol.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(pol.Results))
		}
	})

	t.Run("adds new result when policy differs", func(t *testing.T) {
		t.Parallel()
		pol := &openreportsv1alpha1.Report{
			Results: []openreportsv1alpha1.ReportResult{
				{
					Source: "test-source",
					Policy: "policy-1",
					Rule:   "test-rule",
				},
			},
		}
		newResult := openreportsv1alpha1.ReportResult{
			Source: "test-source",
			Policy: "policy-2",
			Rule:   "test-rule",
		}

		addResult(pol, newResult)

		if len(pol.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(pol.Results))
		}
	})

	t.Run("adds new result when rule differs", func(t *testing.T) {
		t.Parallel()
		pol := &openreportsv1alpha1.Report{
			Results: []openreportsv1alpha1.ReportResult{
				{
					Source: "test-source",
					Policy: "test-policy",
					Rule:   "rule-1",
				},
			},
		}
		newResult := openreportsv1alpha1.ReportResult{
			Source: "test-source",
			Policy: "test-policy",
			Rule:   "rule-2",
		}

		addResult(pol, newResult)

		if len(pol.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(pol.Results))
		}
	})

	t.Run("increments fail summary for each add", func(t *testing.T) {
		t.Parallel()
		pol := &openreportsv1alpha1.Report{}
		result := openreportsv1alpha1.ReportResult{
			Source:     "test-source",
			Policy:     "test-policy",
			Rule:       "test-rule",
			Properties: map[string]string{},
		}

		addResult(pol, result)
		addResult(pol, result) // same result, should update and increment
		newResult := openreportsv1alpha1.ReportResult{
			Source:     "other-source",
			Policy:     "test-policy",
			Rule:       "test-rule",
			Properties: map[string]string{},
		}
		addResult(pol, newResult) // different source, should add new

		if pol.Summary.Fail != 3 {
			t.Errorf("expected summary fail count 3, got %d", pol.Summary.Fail)
		}
	})
}

func TestMergeProperties(t *testing.T) {
	t.Parallel()
	t.Run("increments count when old has count", func(t *testing.T) {
		t.Parallel()
		old := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				propCount: "10",
			},
		}
		new := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				"protocol": "TCP",
			},
		}

		result := mergeProperties(old, new)

		if result[propCount] != "11" {
			t.Errorf("expected count 11, got %q", result[propCount])
		}
		if result["protocol"] != "TCP" {
			t.Errorf("expected protocol 'TCP', got %q", result["protocol"])
		}
	})

	t.Run("starts count at 1 when old has no count", func(t *testing.T) {
		t.Parallel()
		old := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{},
		}
		new := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				"protocol": "TCP",
			},
		}

		result := mergeProperties(old, new)

		if result[propCount] != "1" {
			t.Errorf("expected count 1, got %q", result[propCount])
		}
	})

	t.Run("starts count at 1 when count is invalid", func(t *testing.T) {
		t.Parallel()
		old := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				propCount: "not-a-number",
			},
		}
		new := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{},
		}

		result := mergeProperties(old, new)

		if result[propCount] != "1" {
			t.Errorf("expected count 1, got %q", result[propCount])
		}
	})

	t.Run("preserves created time from old report", func(t *testing.T) {
		t.Parallel()
		old := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				propCount:       "5",
				PropertyCreated: "2024-01-01T00:00:00Z",
			},
		}
		new := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				PropertyCreated: "2024-12-31T23:59:59Z",
				"protocol":      "TCP",
			},
		}

		result := mergeProperties(old, new)

		if result[PropertyCreated] != "2024-01-01T00:00:00Z" {
			t.Errorf("expected created time preserved, got %q", result[PropertyCreated])
		}
		if result["protocol"] != "TCP" {
			t.Errorf("expected protocol 'TCP', got %q", result["protocol"])
		}
	})

	t.Run("sets created time from new when old has none", func(t *testing.T) {
		t.Parallel()
		old := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				propCount: "5",
			},
		}
		new := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				PropertyCreated: "2024-12-31T23:59:59Z",
			},
		}

		result := mergeProperties(old, new)

		if result[PropertyCreated] != "2024-12-31T23:59:59Z" {
			t.Errorf("expected created time from new, got %q", result[PropertyCreated])
		}
	})

	t.Run("merges all properties from new into old", func(t *testing.T) {
		t.Parallel()
		old := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				propCount:       "3",
				PropertyCreated: "2024-01-01T00:00:00Z",
				"protocol":      "TCP",
			},
		}
		new := openreportsv1alpha1.ReportResult{
			Properties: map[string]string{
				PropertyUpdated: "2024-12-31T23:59:59Z",
				"port":          "443",
			},
		}

		result := mergeProperties(old, new)

		if result[propCount] != "4" {
			t.Errorf("expected count 4, got %q", result[propCount])
		}
		if result[PropertyCreated] != "2024-01-01T00:00:00Z" {
			t.Errorf("expected created preserved, got %q", result[PropertyCreated])
		}
		if result[PropertyUpdated] != "2024-12-31T23:59:59Z" {
			t.Errorf("expected updated from new, got %q", result[PropertyUpdated])
		}
		if result["protocol"] != "TCP" {
			t.Errorf("expected protocol preserved, got %q", result["protocol"])
		}
		if result["port"] != "443" {
			t.Errorf("expected port from new, got %q", result["port"])
		}
	})
}
