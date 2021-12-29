// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sampling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/model/pdata"
	"go.uber.org/zap"
)

var tID = pdata.NewTraceID([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})
var sID = pdata.NewSpanID([8]byte{0, 1, 2, 3, 4, 5, 6, 7})

func TestEvaluate_And(t *testing.T) {
	assert := assert.New(t)
	newnop := zap.NewNop()
	str := NewStringAttributeFilter(newnop, "http.method", []string{"GET"}, false, -1, false)
	lat := NewLatency(newnop, 1100)
	and := NewAndPolicy(newnop, []PolicyEvaluator{str, lat})
	trace := createTraceWithAttributeAndDuration("http.method", "GET", 2*time.Second)

	decision, err := and.Evaluate(tID, trace)

	assert.Nil(err)
	assert.Equal(Sampled, decision)
}

//The rate limiter is the only stateful policy to date
func TestEvaluateRatelimit_And(t *testing.T) {
	newnop := zap.NewNop()
	assert := assert.New(t)
	status, _ := NewStatusCodeFilter(newnop, []string{"ERROR"})

	ratelimit := NewRateLimiting(newnop, 2) //with that we can sample 1 span per second

	and := NewAndPolicy(newnop, []PolicyEvaluator{status, ratelimit})

	okTrace := createTraceWithStatusCode(pdata.StatusCodeOk)
	errTrace := createTraceWithStatusCode(pdata.StatusCodeError)
	errTrace2 := createTraceWithStatusCode(pdata.StatusCodeError)

	//if we would activate the rate limiter to make a sampling decision in
	//the evaluation of the first trace, we couldn't sample the second trace
	okDecision, okErr := and.Evaluate(tID, okTrace)
	errDecision, errErr := and.Evaluate(tID, errTrace)
	//here, the number of spans to be sampled by the rate limiter should be used up
	errDecision2, errErr2 := and.Evaluate(tID, errTrace2)

	assert.Nil(okErr)
	assert.Nil(errErr)
	assert.Nil(errErr2)
	assert.Equal(NotSampled, okDecision)
	assert.Equal(Sampled, errDecision)
	assert.Equal(NotSampled, errDecision2)
}

func TestEvaluateNotSampled_And(t *testing.T) {
	newnop := zap.NewNop()
	assert := assert.New(t)
	str := NewStringAttributeFilter(newnop, "http.server_name", []string{"localhost"}, false, -1, false)
	lat := NewLatency(newnop, 1100)
	and := NewAndPolicy(newnop, []PolicyEvaluator{str, lat})
	trace := createTraceWithAttributeAndDuration("http.method", "GET", 2*time.Second)

	decision, err := and.Evaluate(tID, trace)

	assert.Nil(err)
	assert.Equal(NotSampled, decision)
}

func TestOnLateArrivingSpans_And(t *testing.T) {
	newnop := zap.NewNop()
	numeric := NewNumericAttributeFilter(newnop, "Key", 50, 100)
	probabilistic := NewProbabilisticSampler(newnop, "", 0.1)
	and := NewAndPolicy(newnop, []PolicyEvaluator{numeric, probabilistic})
	err := and.OnLateArrivingSpans(NotSampled, nil)
	assert.Nil(t, err)
}

func createTraceWithAttributeAndDuration(key, value string, duration time.Duration) *TraceData {
	traces, span := createTraceWithSpan()
	span.SetStartTimestamp(pdata.NewTimestampFromTime(
		time.Now(),
	))
	span.SetEndTimestamp(pdata.NewTimestampFromTime(span.StartTimestamp().AsTime().Add(duration)))
	span.Attributes().InsertString(key, value)

	return &TraceData{
		ReceivedBatches: []pdata.Traces{traces},
		SpanCount:       1,
	}
}

func createTraceWithStatusCode(status pdata.StatusCode) *TraceData {
	traces, span := createTraceWithSpan()
	span.Status().SetCode(status)
	return &TraceData{
		ReceivedBatches: []pdata.Traces{traces},
		SpanCount:       1,
	}
}

func createTraceWithSpan() (pdata.Traces, pdata.Span) {
	traces := pdata.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	ils := rs.InstrumentationLibrarySpans().AppendEmpty()
	span := ils.Spans().AppendEmpty()
	span.SetTraceID(tID)
	span.SetSpanID(sID)
	return traces, span
}
