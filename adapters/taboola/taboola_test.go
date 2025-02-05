package taboola

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderTaboola,
		config.Adapter{
			Endpoint: "http://{{.MediaType}}.whatever.com/{{.GvlID}}/{{.PublisherID}}",
		},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 12, DataCenter: "2"},
	)
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	adapterstest.RunJSONBidderTest(t, "taboolatest", bidder)
}

// TestEmptyExternalUrl checks if gvlID remains empty when GvlID=0
func TestEmptyExternalUrl(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderTaboola,
		config.Adapter{Endpoint: "http://whatever.com"},
		config.Server{},
	)
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	bidderTaboola := bidder.(*adapter)
	assert.Equal(t, "", bidderTaboola.gvlID, "Expected empty gvlID when GvlID=0")
}

// TestNoEids ensures that if user.ext has no taboola.com eids, we don't set BuyerUID.
func TestNoEids(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderTaboola,
		config.Adapter{Endpoint: "http://whatever.com"},
		config.Server{},
	)
	assert.NoError(t, buildErr)
	w, h := int64(300), int64(250)

	// Minimal request with no user or user.ext
	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID:     "imp1",
				Banner: &openrtb2.Banner{W: &w, H: &h},
				Ext:    json.RawMessage(`{"bidder":{"tagid": "example-tag"}}`),
			},
		},
		// No user at all -> no eids
	}

	// MakeRequests
	requests, errs := bidder.MakeRequests(req, nil)
	assert.Empty(t, errs, "Expected no errors")
	assert.NotEmpty(t, requests, "Should produce at least one request")

	// Parse the request body to confirm BuyerUID is not set
	for _, r := range requests {
		var parsed openrtb2.BidRequest
		err := json.Unmarshal(r.Body, &parsed)
		assert.NoError(t, err)

		if parsed.User != nil {
			assert.Empty(t, parsed.User.BuyerUID, "Expected BuyerUID to remain empty when no taboola.com eids present")
		}
	}
}

// TestWithTaboolaEid ensures that if user.ext.eids includes source=taboola.com,
// we set user.BuyerUID to that ID.
func TestWithTaboolaEid(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderTaboola,
		config.Adapter{Endpoint: "http://whatever.com"},
		config.Server{},
	)
	assert.NoError(t, buildErr)

	// Create user.ext that has an eid with source="taboola.com"
	eidsJSON := `{
      "eids": [
        {
          "source": "taboola.com",
          "uids": [{"id": "taboola-user-123"}]
        },
        {
          "source": "other.com",
          "uids": [{"id": "not-used"}]
        }
      ]
    }`
	userExt := json.RawMessage(eidsJSON)
	w, h := int64(300), int64(250)
	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID:     "imp1",
				Banner: &openrtb2.Banner{W: &w, H: &h},
				Ext:    json.RawMessage(`{"bidder":{"tagid": "example-tag"}}`),
			},
		},
		User: &openrtb2.User{
			Ext: userExt,
		},
	}

	// MakeRequests
	requests, errs := bidder.MakeRequests(req, nil)
	assert.Empty(t, errs, "Expected no errors with valid eids")
	assert.NotEmpty(t, requests, "Should produce at least one request")

	// Check that user.BuyerUID was set to "taboola-user-123"
	for _, r := range requests {
		var parsed openrtb2.BidRequest
		err := json.Unmarshal(r.Body, &parsed)
		assert.NoError(t, err)

		if parsed.User != nil {
			assert.Equal(t, "taboola-user-123", parsed.User.BuyerUID,
				"Expected BuyerUID to be set from taboola.com eids ID")
		}
	}
}
