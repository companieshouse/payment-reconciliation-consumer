package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitHealthCheck(t *testing.T) {
	Convey("Check 200 Response for Healthcheck", t, func() {
		req, err := http.NewRequest("GET", "/payment-reconciliation-consumer/healthcheck", nil)
		So(err, ShouldBeNil)
		response := httptest.NewRecorder()

		HealthCheck(response, req)
		So(response.Code, ShouldEqual, 200)
	})
}
