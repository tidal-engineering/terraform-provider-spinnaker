package spinnaker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	gate "github.com/spinnaker/spin/cmd/gateclient"
	gateapi "github.com/spinnaker/spin/gateapi"
)

// A pipeline deleted outside Terraform, for example with its application, makes Gate return 404.
func gateReturning404(t *testing.T) gateConfig {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/applications/gone/pipelineConfigs/deploy" {
			t.Errorf("unexpected request path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	cfg := gateapi.NewConfiguration()
	cfg.BasePath = server.URL
	return gateConfig{client: &gate.GatewayClient{APIClient: gateapi.NewAPIClient(cfg), Context: context.Background()}}
}

func pipelineInState(t *testing.T) *schema.ResourceData {
	data := schema.TestResourceDataRaw(t, resourcePipeline().Schema, map[string]interface{}{
		"application": "gone",
		"name":        "deploy",
		"pipeline":    "{}",
	})
	data.SetId("5813a9f1-517d-4b17-b986-1d8d7b0e707f")
	return data
}

func TestResourcePipelineExists_pipelineGone(t *testing.T) {
	exists, err := resourcePipelineExists(pipelineInState(t), gateReturning404(t))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatal("expected the pipeline not to exist")
	}
}

func TestResourcePipelineRead_pipelineGone(t *testing.T) {
	data := pipelineInState(t)
	if err := resourcePipelineRead(data, gateReturning404(t)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if data.Id() != "" {
		t.Fatalf("expected the pipeline to be removed from state, id is %q", data.Id())
	}
}

func TestDatasourcePipelineRead_pipelineGone(t *testing.T) {
	data := schema.TestResourceDataRaw(t, datasourcePipeline().Schema, map[string]interface{}{
		"application": "gone",
		"name":        "deploy",
	})
	err := datasourcePipeline().Read(data, gateReturning404(t))
	if err == nil || err.Error() != "NoSuchEntityException" {
		t.Fatalf("expected NoSuchEntityException, got %v", err)
	}
}
