package glflake

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

var gf *glflake

var startTime int64
var machineID int64
var serviceID int64

func init() {
	var st Settings
	st.StartTime = time.Now()
	st.MachineID = func() (uint16, error) {
		return 1, nil
	}

	gf = NewGlflake(st)
	if gf == nil {
		panic("glflake not created")
	}

	startTime = toGlflakeTime(st.StartTime)
	machineID = 1
	serviceID = 2
}

func nextID(t *testing.T) ID {
	id, err := gf.NextID()
	if err != nil {
		t.Fatal("id not generated:", err)
	}
	return id
}

func testReset(t *testing.T) {
	gf.startTime = 0
	gf.elapsedTime = 0
}

func TestGlflakeOnce(t *testing.T) {
	sleepTime := int64(50)
	time.Sleep(time.Duration(sleepTime) * 10 * time.Millisecond)

	id := nextID(t)
	parts := Decompose(id)

	actualMSB := parts["msb"]
	if actualMSB != 0 {
		t.Errorf("unexpected msb: %d", actualMSB)
	}

	actualTime := parts["time"]
	if actualTime < sleepTime || actualTime > sleepTime+1 {
		t.Errorf("unexpected time: %d", actualTime)
	}

	actualMachineID := parts["machine-id"]
	if actualMachineID != machineID {
		t.Errorf("unexpected machine id: %d", actualMachineID)
	}

	actualSequence := parts["sequence"]
	if actualSequence != 0 {
		t.Errorf("unexpected sequence: %d", actualSequence)
	}

	fmt.Println("glflake id:", id)
	fmt.Println("decompose:", parts)
	fmt.Println("hex of id:", fmt.Sprintf("%02X", id))
}

func currentTime() int64 {
	return toGlflakeTime(time.Now())
}

func TestGlflakeFor10Sec(t *testing.T) {
	var numID uint32
	var lastID int64
	var maxSequence int64

	initial := currentTime()
	current := initial
	for current-initial < 1000 {
		id := nextID(t)
		parts := Decompose(id)
		numID++

		if int64(id) <= lastID {
			t.Fatal("duplicated id")
		}
		lastID = int64(id)

		current = currentTime()

		actualMSB := parts["msb"]
		if actualMSB != 0 {
			t.Errorf("unexpected msb: %d", actualMSB)
		}

		actualTime := int64(parts["time"])
		overtime := startTime + actualTime - current
		if overtime > 0 {
			t.Errorf("unexpected overtime: %d", overtime)
		}

		actualMachineID := parts["machine-id"]
		if actualMachineID != machineID {
			t.Errorf("unexpected machine id: %d", actualMachineID)
		}

		actualSequence := parts["sequence"]
		if maxSequence < actualSequence {
			maxSequence = actualSequence
		}
	}

	if maxSequence != 1<<BitLenSequence-1 {
		t.Errorf("unexpected max sequence: %d", maxSequence)
	}
	fmt.Println("max sequence:", maxSequence)
	fmt.Println("number of id:", numID)
}

func TestGlflakeInParallel(t *testing.T) {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	fmt.Println("number of cpu:", numCPU)

	consumer := make(chan ID)

	const numID = 10000
	generate := func() {
		for i := 0; i < numID; i++ {
			consumer <- nextID(t)
		}
	}

	const numGenerator = 10
	for i := 0; i < numGenerator; i++ {
		go generate()
	}

	set := make(map[ID]struct{})
	for i := 0; i < numID*numGenerator; i++ {
		id := <-consumer
		if _, ok := set[id]; ok {
			t.Fatal("duplicated id")
		}
		set[id] = struct{}{}
	}
	fmt.Println("number of id:", len(set))
}

func TestNilglflake(t *testing.T) {
	var startInFuture Settings
	startInFuture.StartTime = time.Now().Add(time.Duration(1) * time.Minute)
	if NewGlflake(startInFuture) != nil {
		t.Errorf("glflake starting in the future")
	}

	var noMachineID Settings
	noMachineID.MachineID = func() (uint16, error) {
		return 0, fmt.Errorf("no machine id")
	}
	if NewGlflake(noMachineID) != nil {
		t.Errorf("glflake with no machine id")
	}

	var invalidMachineID Settings
	invalidMachineID.CheckMachineID = func(uint16) bool {
		return false
	}
	if NewGlflake(invalidMachineID) != nil {
		t.Errorf("glflake with invalid machine id")
	}
}

func pseudoSleep(period time.Duration) {
	gf.startTime -= int64(period)
}

func TestNextIDError(t *testing.T) {
	year := time.Duration(365*24) * time.Hour / glflakeTimeUnit
	pseudoSleep(time.Duration(174) * year)
	nextID(t)

	pseudoSleep(time.Duration(1) * year)
	_, err := gf.NextID()
	if err == nil {
		t.Errorf("time is not over")
	}
}
