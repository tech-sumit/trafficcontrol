/*

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/testing/api/assert"
	"github.com/apache/trafficcontrol/traffic_ops/testing/api/utils"
)

const queryParamFormat = "?profileId=%s&parameterId=%s"

func TestProfileParameters(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Profiles, ProfileParameters}, func() {

		currentTime := time.Now().UTC().Add(-15 * time.Second)
		tomorrow := currentTime.AddDate(0, 0, 1).Format(time.RFC1123)

		methodTests := utils.V3TestCase{
			"GET": {
				"NOT MODIFIED when NO CHANGES made": {
					ClientSession:  TOSession,
					RequestHeaders: http.Header{rfc.IfModifiedSince: {tomorrow}},
					Expectations:   utils.CkRequest(utils.NoError(), utils.HasStatus(http.StatusNotModified)),
				},
				"OK when VALID request": {
					ClientSession: TOSession,
					RequestParams: url.Values{
						"profileId":   {strconv.Itoa(GetProfileID(t, "RASCAL1")())},
						"parameterId": {strconv.Itoa(GetParameterID(t, "peers.polling.interval", "rascal-config.txt", "60")())}},
					Expectations: utils.CkRequest(utils.NoError(), utils.HasStatus(http.StatusOK)),
				},
			},
			"POST": {
				"OK when MULTIPLE PARAMETERS": {
					ClientSession: TOSession,
					RequestBody: map[string]interface{}{
						"profileParameters": []map[string]interface{}{
							{
								"profileId":   GetProfileID(t, "MID1")(),
								"parameterId": GetParameterID(t, "CONFIG proxy.config.admin.user_id", "records.config", "STRING ats")(),
							},
							{
								"profileId":   GetProfileID(t, "MID2")(),
								"parameterId": GetParameterID(t, "CONFIG proxy.config.admin.user_id", "records.config", "STRING ats")(),
							},
						},
					},
					Expectations: utils.CkRequest(utils.NoError(), utils.HasStatus(http.StatusOK)),
				},
				"BAD REQUEST when INVALID PROFILEID and PARAMETERID": {
					ClientSession: TOSession,
					RequestBody: map[string]interface{}{
						"profileId":   0,
						"parameterId": 0,
					},
					Expectations: utils.CkRequest(utils.HasError(), utils.HasStatus(http.StatusBadRequest)),
				},
				"BAD REQUEST when MISSING PROFILEID field": {
					ClientSession: TOSession,
					RequestBody: map[string]interface{}{
						"parameterId": GetParameterID(t, "health.threshold.queryTime", "rascal.properties", "1000")(),
					},
					Expectations: utils.CkRequest(utils.HasError(), utils.HasStatus(http.StatusBadRequest)),
				},
				"BAD REQUEST when MISSING PARAMETERID field": {
					ClientSession: TOSession,
					RequestBody: map[string]interface{}{
						"profileId": GetProfileID(t, "EDGE2")(),
					},
					Expectations: utils.CkRequest(utils.HasError(), utils.HasStatus(http.StatusBadRequest)),
				},
				"BAD REQUEST when EMPTY BODY": {
					ClientSession: TOSession,
					RequestBody:   map[string]interface{}{},
					Expectations:  utils.CkRequest(utils.HasError(), utils.HasStatus(http.StatusBadRequest)),
				},
				"BAD REQUEST when ALREADY EXISTS": {
					ClientSession: TOSession,
					RequestBody: map[string]interface{}{
						"profileId":   GetProfileID(t, "EDGE1")(),
						"parameterId": GetParameterID(t, "health.threshold.availableBandwidthInKbps", "rascal.properties", ">1750000")(),
					},
					Expectations: utils.CkRequest(utils.HasError(), utils.HasStatus(http.StatusBadRequest)),
				},
			},
			"DELETE": {
				"OK when VALID request": {
					EndpointId:    GetProfileID(t, "ATS_EDGE_TIER_CACHE"),
					ClientSession: TOSession,
					RequestParams: url.Values{
						"parameterId": {strconv.Itoa(GetParameterID(t, "location", "set_dscp_37.config", "/etc/trafficserver/dscp")())},
					},
					Expectations: utils.CkRequest(utils.NoError(), utils.HasStatus(http.StatusOK)),
				},
			},
		}

		for method, testCases := range methodTests {
			t.Run(method, func(t *testing.T) {
				for name, testCase := range testCases {
					profileParameter := tc.ProfileParameter{}
					profileParameters := []tc.ProfileParameter{}

					if testCase.RequestBody != nil {
						if profileParams, ok := testCase.RequestBody["profileParameters"]; ok {
							dat, err := json.Marshal(profileParams)
							assert.NoError(t, err, "Error occurred when marshalling request body: %v", err)
							err = json.Unmarshal(dat, &profileParameters)
							assert.NoError(t, err, "Error occurred when unmarshalling request body: %v", err)
						}
						dat, err := json.Marshal(testCase.RequestBody)
						assert.NoError(t, err, "Error occurred when marshalling request body: %v", err)
						err = json.Unmarshal(dat, &profileParameter)
						assert.NoError(t, err, "Error occurred when unmarshalling request body: %v", err)
					}

					switch method {
					case "GET":
						t.Run(name, func(t *testing.T) {
							if testCase.RequestParams == nil {
								resp, reqInf, err := testCase.ClientSession.GetProfileParametersWithHdr(testCase.RequestHeaders)
								for _, check := range testCase.Expectations {
									check(t, reqInf, resp, tc.Alerts{}, err)
								}
							} else {
								queryParams := fmt.Sprintf(queryParamFormat, testCase.RequestParams["profileId"][0], testCase.RequestParams["parameterId"][0])
								resp, reqInf, err := testCase.ClientSession.GetProfileParameterByQueryParamsWithHdr(queryParams, testCase.RequestHeaders)
								for _, check := range testCase.Expectations {
									check(t, reqInf, resp, tc.Alerts{}, err)
								}
							}
						})
					case "POST":
						t.Run(name, func(t *testing.T) {
							if len(profileParameters) == 0 {
								alerts, reqInf, err := testCase.ClientSession.CreateProfileParameter(profileParameter)
								for _, check := range testCase.Expectations {
									check(t, reqInf, nil, alerts, err)
								}
							} else {
								alerts, reqInf, err := testCase.ClientSession.CreateMultipleProfileParameters(profileParameters)
								for _, check := range testCase.Expectations {
									check(t, reqInf, nil, alerts, err)
								}
							}
						})
					case "DELETE":
						t.Run(name, func(t *testing.T) {
							parameterId, _ := strconv.Atoi(testCase.RequestParams["parameterId"][0])
							alerts, reqInf, err := testCase.ClientSession.DeleteParameterByProfileParameter(testCase.EndpointId(), parameterId)
							for _, check := range testCase.Expectations {
								check(t, reqInf, nil, alerts, err)
							}
						})
					}
				}
			})
		}
	})
}

func CreateTestProfileParameters(t *testing.T) {
	for _, profile := range testData.Profiles {
		profileID := GetProfileID(t, profile.Name)()

		for _, parameter := range profile.Parameters {
			assert.RequireNotNil(t, parameter.Name, "Expected parameter name to not be nil.")
			assert.RequireNotNil(t, parameter.Value, "Expected parameter value to not be nil.")
			assert.RequireNotNil(t, parameter.ConfigFile, "Expected parameter configFile to not be nil.")

			getParameter, _, err := TOSession.GetParameterByNameAndConfigFileAndValueWithHdr(*parameter.Name, *parameter.ConfigFile, *parameter.Value, nil)
			assert.RequireNoError(t, err, "Could not get Parameter %s: %v", *parameter.Name, err)
			if len(getParameter) == 0 {
				alerts, _, err := TOSession.CreateParameter(tc.Parameter{Name: *parameter.Name, Value: *parameter.Value, ConfigFile: *parameter.ConfigFile})
				assert.RequireNoError(t, err, "Could not create Parameter %s: %v - alerts: %+v", parameter.Name, err, alerts.Alerts)
				getParameter, _, err = TOSession.GetParameterByNameAndConfigFileAndValueWithHdr(*parameter.Name, *parameter.ConfigFile, *parameter.Value, nil)
				assert.RequireNoError(t, err, "Could not get Parameter %s: %v", *parameter.Name, err)
				assert.RequireNotEqual(t, 0, len(getParameter), "Could not get parameter %s: not found", *parameter.Name)
			}
			profileParameter := tc.ProfileParameter{ProfileID: profileID, ParameterID: getParameter[0].ID}
			alerts, _, err := TOSession.CreateProfileParameter(profileParameter)
			assert.NoError(t, err, "Could not associate Parameter %s with Profile %s: %v - alerts: %+v", parameter.Name, profile.Name, err, alerts.Alerts)
		}
	}
}

func DeleteTestProfileParameters(t *testing.T) {
	profileParameters, _, err := TOSession.GetProfileParametersWithHdr(nil)
	assert.NoError(t, err, "Cannot get Profile Parameters: %v - alerts: %+v", err)

	for _, profileParameter := range profileParameters {
		alerts, _, err := TOSession.DeleteParameterByProfileParameter(GetProfileID(t, profileParameter.Profile)(), profileParameter.ParameterID)
		assert.NoError(t, err, "Unexpected error deleting Profile Parameter: Profile: '%s' Parameter: %s: %v - alerts: %+v", profileParameter.Profile, profileParameter.Parameter, err, alerts.Alerts)
	}
	// Retrieve the Profile Parameters to see if it got deleted
	getProfileParameter, _, err := TOSession.GetProfileParametersWithHdr(nil)
	assert.NoError(t, err, "Error getting Profile Parameters after deletion: %v - alerts: %+v", err)
	assert.Equal(t, 0, len(getProfileParameter), "Expected Profile Parameters to be deleted, but %d were found in Traffic Ops", len(getProfileParameter))
}
