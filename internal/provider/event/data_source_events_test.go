package event

import (
	"testing"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// evAttrString reads a string attribute from a types.Object.
func evAttrString(obj types.Object, key string) string {
	v, ok := obj.Attributes()[key]
	if !ok {
		return ""
	}
	sv, ok := v.(types.String)
	if !ok {
		return ""
	}
	return sv.ValueString()
}

// evAttrBool reads a bool attribute from a types.Object.
func evAttrBool(obj types.Object, key string) bool {
	v, ok := obj.Attributes()[key]
	if !ok {
		return false
	}
	bv, ok := v.(types.Bool)
	if !ok {
		return false
	}
	return bv.ValueBool()
}

// ---- eventToObject ----

func TestEventToObject_BasicFields(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	start := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 1, 11, 0, 0, 0, time.UTC)
	subseg := true
	e := models.Event{
		EventID:            "ev-1",
		EventName:          "my-event",
		StreamID:           "s-1",
		Active:             true,
		Ended:              false,
		StartTime:          &start,
		EndTime:            &end,
		SubsegmentClipping: &subseg,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	obj, diags := eventToObject(e)
	if diags.HasError() {
		t.Fatalf("eventToObject() returned errors: %v", diags)
	}

	strCases := map[string]string{
		"id":         "ev-1",
		"event_name": "my-event",
		"stream_id":  "s-1",
		"start_time": start.Format(time.RFC3339),
		"end_time":   end.Format(time.RFC3339),
		"created_at": now.Format(time.RFC3339),
		"updated_at": now.Format(time.RFC3339),
	}
	for key, want := range strCases {
		if got := evAttrString(obj, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if !evAttrBool(obj, "active") {
		t.Error("active should be true")
	}
	if evAttrBool(obj, "ended") {
		t.Error("ended should be false")
	}
	if !evAttrBool(obj, "subsegment_clipping") {
		t.Error("subsegment_clipping should be true")
	}
}

func TestEventToObject_NilStartEndTimeProduceEmptyStrings(t *testing.T) {
	e := models.Event{EventID: "ev-1"} // nil StartTime and EndTime
	obj, diags := eventToObject(e)
	if diags.HasError() {
		t.Fatalf("eventToObject() returned errors: %v", diags)
	}
	if got := evAttrString(obj, "start_time"); got != "" {
		t.Errorf("start_time should be empty for nil, got %q", got)
	}
	if got := evAttrString(obj, "end_time"); got != "" {
		t.Errorf("end_time should be empty for nil, got %q", got)
	}
}

func TestEventToObject_NilSubsegmentClippingDefaultsFalse(t *testing.T) {
	e := models.Event{EventID: "ev-1"} // nil SubsegmentClipping
	obj, diags := eventToObject(e)
	if diags.HasError() {
		t.Fatalf("eventToObject() returned errors: %v", diags)
	}
	if evAttrBool(obj, "subsegment_clipping") {
		t.Error("subsegment_clipping should default to false when nil")
	}
}

func TestEventToObject_ZeroTimestampsProduceEmptyStrings(t *testing.T) {
	e := models.Event{EventID: "ev-1"} // zero CreatedAt / UpdatedAt
	obj, diags := eventToObject(e)
	if diags.HasError() {
		t.Fatalf("eventToObject() returned errors: %v", diags)
	}
	if got := evAttrString(obj, "created_at"); got != "" {
		t.Errorf("created_at should be empty for zero time, got %q", got)
	}
	if got := evAttrString(obj, "updated_at"); got != "" {
		t.Errorf("updated_at should be empty for zero time, got %q", got)
	}
}

func TestEventToObject_ContainsAllExpectedKeys(t *testing.T) {
	e := models.Event{EventID: "ev-1"}
	obj, diags := eventToObject(e)
	if diags.HasError() {
		t.Fatalf("eventToObject() returned errors: %v", diags)
	}
	for key := range eventAttrTypes {
		if _, ok := obj.Attributes()[key]; !ok {
			t.Errorf("missing attribute %q in object", key)
		}
	}
}

func TestEventToObject_SubsegmentClippingFalse(t *testing.T) {
	f := false
	e := models.Event{EventID: "ev-1", SubsegmentClipping: &f}
	obj, diags := eventToObject(e)
	if diags.HasError() {
		t.Fatalf("eventToObject() returned errors: %v", diags)
	}
	if evAttrBool(obj, "subsegment_clipping") {
		t.Error("subsegment_clipping should be false")
	}
}
