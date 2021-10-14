// Package glflake implements glflake, duoxieyun distributed unique ID generator inspired by Twitter's Snowflake.
//
// +------------------------------------------------------------------------+
// | 1 Bit Unused | 39 Bit Timestamp | 16 Bit MachineID | 8 Bit Sequence ID |
// +------------------------------------------------------------------------+
//
// 39 bits for time in units of 10 msec (178 years)
// 16 bits for a machine id (65536 nodes, distributed machines)
//  8 bits for a sequence number (0 ~ 255))
package glflake

import (
	"errors"
	"sync"
	"time"
)

// These constants are the bit lengths of glflake ID parts.
const (
	BitLenTime      = 39 // bit length of time
	BitLenMachineID = 16 // bit length of machineID
	BitLenSequence  = 8  // bit length of sequence number
)

// Settings configures glflake:
//
// StartTime is the time since which the glflake time is defined as the elapsed time.
// If StartTime is 0, the start time of the glflake is set to "2021-10-01 00:00:00 +0000 UTC".
// If StartTime is ahead of the current time, glflake is not created.
//
// MachineID returns the unique ID of the glflake instance.
// If MachineID returns an error, glflake is not created.
// If MachineID is nil, default MachineID(0) is used.
//
// CheckMachineID validates the uniqueness of the machine ID.
// If CheckMachineID returns false, glflake is not created.
// If CheckMachineID is nil, no validation is done.
type Settings struct {
	StartTime      time.Time
	MachineID      func() (uint16, error)
	CheckMachineID func(uint16) bool
}

// Init set default MachineID & ServiceID
func (s *Settings) Init(mID uint16) {
	if s != nil {
		s.MachineID = func() (uint16, error) {
			return mID, nil
		}
	}
}

// StartTimeSet set start time
func (s *Settings) StartTimeSet(t time.Time) {
	if s != nil {
		s.StartTime = t
	}
}

// glflake is a distributed unique ID generator.
type glflake struct {
	mutex       *sync.Mutex
	startTime   int64
	elapsedTime int64
	machineID   uint16
	sequence    uint16
}

// NewGlflake returns a new glflake configured with the given Settings.
// NewGlflake returns nil in the following cases:
// - Settings.StartTime is ahead of the current time.
// - Settings.MachineID returns an error.
// - Settings.CheckMachineID returns false.
func NewGlflake(st Settings) *glflake {
	gf := new(glflake)
	gf.mutex = new(sync.Mutex)
	gf.sequence = uint16(1<<BitLenSequence - 1)

	if st.StartTime.After(time.Now()) {
		return nil
	}
	if st.StartTime.IsZero() {
		gf.startTime = toGlflakeTime(time.Date(2021, 10, 1, 0, 0, 0, 0, time.UTC))
	} else {
		gf.startTime = toGlflakeTime(st.StartTime)
	}

	var err error
	if st.MachineID == nil {
		gf.machineID, err = lower16BitPrivateIP()
	} else {
		gf.machineID, err = st.MachineID()
	}
	if err != nil ||
		(st.CheckMachineID != nil && !st.CheckMachineID(gf.machineID)) {
		return nil
	}

	return gf
}

// NextID generates a next unique ID.
// After the glflake time overflows, NextID returns an error.
func (gf *glflake) NextID() (ID, error) {
	const maskSequence = uint16(1<<BitLenSequence - 1)

	gf.mutex.Lock()
	defer gf.mutex.Unlock()

	current := currentElapsedTime(gf.startTime)
	if gf.elapsedTime < current {
		gf.elapsedTime = current
		gf.sequence = 0
	} else { // gf.elapsedTime >= current
		gf.sequence = (gf.sequence + 1) & maskSequence
		if gf.sequence == 0 { // overflow
			gf.elapsedTime++
			overtime := gf.elapsedTime - current
			time.Sleep(sleepTime((overtime)))
		}
	}

	return gf.toID()
}

const glflakeTimeUnit = 1e7 // nsec, i.e. 10 msec

func toGlflakeTime(t time.Time) int64 {
	return t.UTC().UnixNano() / glflakeTimeUnit
}

func currentElapsedTime(startTime int64) int64 {
	return toGlflakeTime(time.Now()) - startTime
}

func sleepTime(overtime int64) time.Duration {
	return time.Duration(overtime)*10*time.Millisecond -
		time.Duration(time.Now().UTC().UnixNano()%glflakeTimeUnit)*time.Nanosecond
}

func (gf *glflake) toID() (ID, error) {
	if gf.elapsedTime >= 1<<BitLenTime {
		return 0, errors.New("over the time limit")
	}

	return ID(int64(gf.elapsedTime)<<(BitLenMachineID+BitLenSequence) |
		int64(gf.machineID)<<(BitLenSequence) |
		int64(gf.sequence)), nil
}

// Decompose returns a set of glflake ID parts.
func Decompose(id ID) map[string]int64 {
	const maskMachineID = int64((1<<BitLenMachineID - 1) << BitLenSequence)
	const maskSequence = int64(1<<BitLenSequence - 1)

	msb := int64(id) >> 63
	time := int64(id) >> (BitLenMachineID + BitLenSequence)
	machineID := (int64(id) & maskMachineID) >> BitLenSequence
	sequence := (int64(id) & maskSequence)
	return map[string]int64{
		"id":         int64(id),
		"msb":        msb,
		"time":       time,
		"machine-id": machineID,
		"sequence":   sequence,
	}
}
