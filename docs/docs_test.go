package docs

import (
	"testing"

	"github.com/swaggo/swag"
)

func TestSwaggerInfo_Registered(t *testing.T) {
	s := swag.GetSwagger(SwaggerInfo.InstanceName())
	if s == nil {
		t.Fatalf("expected swagger spec to be registered")
	}
}
