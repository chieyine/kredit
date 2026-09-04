package businesspolicy

import (
	"encoding/json"
	"kredit/internal/config"
	"testing"
)

func TestPolicyRequiresCompleteValidValuesAndDeploymentCeilings(t *testing.T) {
	d := Defaults(config.Config{})
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{}`, `{"automatic_collection":true}`, `null`} {
		var v Values
		if json.Unmarshal([]byte(bad), &v) == nil {
			t.Fatal("accepted incomplete policy")
		}
	}
	bytes, _ := json.Marshal(d)
	var fields map[string]any
	_ = json.Unmarshal(bytes, &fields)
	fields["max_retries"] = 1.5
	bytes, _ = json.Marshal(fields)
	var v Values
	if json.Unmarshal(bytes, &v) == nil {
		t.Fatal("accepted fractional attempts")
	}
	for _, change := range []func(*Values){func(v *Values) { v.AutomaticRetry = true }, func(v *Values) { v.CorrectionThreshold = 1000001 }, func(v *Values) { v.NoticeHours = 0 }, func(v *Values) { v.BaseFeeBPS = 1001 }, func(v *Values) { v.MaxPrincipal = -1 }} {
		v := d
		change(&v)
		if v.Validate() == nil {
			t.Fatalf("accepted invalid policy: %+v", v)
		}
	}
	cfg := config.Config{PilotMaxPrincipalKobo: 100000, CollectionNoticeMinHours: 24, RealCollections: true, PilotAllowedIndustries: "retail", PilotEnhancedReviewKobo: 50000}
	v = Defaults(cfg)
	if err := v.ValidateDeployment(cfg); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Values){func(v *Values) { v.MaxPrincipal = 0 }, func(v *Values) { v.NoticeHours = 1 }, func(v *Values) { v.AllowedIndustries = "retail,other" }, func(v *Values) { v.EnhancedReview = 0 }} {
		bad := v
		change(&bad)
		if bad.ValidateDeployment(cfg) == nil {
			t.Fatal("deployment boundary bypass")
		}
	}
}
