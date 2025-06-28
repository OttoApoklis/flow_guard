package snowflack

import (
	"fmt"
	"sync"
	"time"
)

var (
	instance *Snowflake
	once     sync.Once
)

// Snowflake struct holds the machine ID and sequence ID
type Snowflake struct {
	machineID     int64 // Machine ID (for example, can be 0-1023)
	dataCenterID  int64 // Data center ID (can be 0-1023)
	sequence      int64 // Sequence number
	lastTimestamp int64 // Last timestamp
	lock          sync.Mutex
}

// Constants for Snowflake bit allocations
const (
	epoch           = int64(1609459200000)               // Epoch (custom starting point, e.g., 2021-01-01)
	machineBits     = uint64(10)                         // Number of bits for machine ID
	dataCenterBits  = uint64(10)                         // Number of bits for data center ID
	sequenceBits    = uint64(12)                         // Number of bits for sequence
	maxMachineID    = int64(-1 ^ (-1 << machineBits))    // Max machine ID
	maxDataCenterID = int64(-1 ^ (-1 << dataCenterBits)) // Max data center ID
	maxSequence     = int64(-1 ^ (-1 << sequenceBits))   // Max sequence ID

	// Shifts for different parts of the Snowflake ID
	machineIDShift    = sequenceBits
	dataCenterIDShift = sequenceBits + machineBits
	timestampShift    = sequenceBits + machineBits + dataCenterBits
)

// NewSnowflake creates a new Snowflake instance with the given machine and data center IDs
func NewSnowflake(machineID, dataCenterID int64) (*Snowflake, error) {
	if machineID < 0 || machineID > maxMachineID {
		return nil, fmt.Errorf("machineID out of range")
	}
	if dataCenterID < 0 || dataCenterID > maxDataCenterID {
		return nil, fmt.Errorf("dataCenterID out of range")
	}

	return &Snowflake{
		machineID:    machineID,
		dataCenterID: dataCenterID,
	}, nil
}

// GenerateID generates a unique Snowflake ID
func (s *Snowflake) GenerateID() int64 {
	s.lock.Lock()
	defer s.lock.Unlock()

	timestamp := time.Now().UnixNano() / int64(time.Millisecond) // Current timestamp in milliseconds
	if timestamp == s.lastTimestamp {
		// Same timestamp, increment sequence
		s.sequence = (s.sequence + 1) & maxSequence
		if s.sequence == 0 {
			// Sequence overflow, wait for next millisecond
			for timestamp <= s.lastTimestamp {
				timestamp = time.Now().UnixNano() / int64(time.Millisecond)
			}
		}
	} else {
		// New timestamp, reset sequence
		s.sequence = 0
	}

	s.lastTimestamp = timestamp

	// Shift the bits and combine them into a unique ID
	id := (timestamp-epoch)<<timestampShift |
		(s.dataCenterID << dataCenterIDShift) |
		(s.machineID << machineIDShift) |
		s.sequence

	return id
}

func GetSnowFlackID() int64 {
	return GetSnowFlack().GenerateID()
}

// GetInstance returns the singleton instance of Singleton
func GetSnowFlack() *Snowflake {
	once.Do(func() {
		instance, _ = NewSnowflake(1, 1)
	})
	return instance
}
