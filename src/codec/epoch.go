package codec

import (
	"encoding/json"
	"time"
)

type EpochTime struct {
	time.Time
}

func (e *EpochTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Unix())
}

func (e *EpochTime) UnmarshalJSON(data []byte) error {
	var epoch float64
	err := json.Unmarshal(data, &epoch)
	if err != nil {
		return err
	}

	e.Time = time.Unix(int64(epoch), 0)
	return nil
}
