package milli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMilli(t *testing.T) {
	currentTime := time.Now()
	currentTimeMilli := Timestamp(currentTime)
	assert.Equal(t, currentTime.Hour(), Time(currentTimeMilli).Hour())
	assert.Equal(t, currentTime.Minute(), Time(currentTimeMilli).Minute())
	expectedYear, expectedMonth, expectedDay := currentTime.Date()
	actualYear, actualMonth, actualDay := Time(currentTimeMilli).Date()
	assert.Equal(t, expectedYear, actualYear)
	assert.Equal(t, expectedMonth, actualMonth)
	assert.Equal(t, expectedDay, actualDay)
}
