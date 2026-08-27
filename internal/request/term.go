package request

import (
	"encoding/json"
	"time"
)

// -----------------------------------------------------------
// 招新换届
var shanghaiLocation, _ = time.LoadLocation("Asia/Shanghai")

type CreateTermReq struct {
	Year        uint64     `json:"year" binding:"required"`
	Type        string     `json:"type" binding:"required"`
	Title       string     `json:"title" binding:"required"`
	EditPeriod  TimePeriod `json:"edit_period" binding:"required"`
	QueryPeriod TimePeriod `json:"query_period" binding:"required"`
}
type UpdateTermReq struct {
	ID          uint64     `json:"id" binding:"required"`
	Title       string     `json:"title" binding:"required"`
	EditPeriod  TimePeriod `json:"edit_period" binding:"required"`
	QueryPeriod TimePeriod `json:"query_period" binding:"required"`
}

type TimePeriod struct {
	StartAt time.Time `json:"start_at" binding:"required"`
	EndAt   time.Time `json:"end_at" binding:"required"`
}

func (t *TimePeriod) UnmarshalJSON(data []byte) error {
	type Alias struct {
		StartAt string `json:"start_at"`
		EndAt   string `json:"end_at"`
	}

	var v Alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	start, err := time.ParseInLocation("2006-01-02", v.StartAt, shanghaiLocation)
	if err != nil {
		return err
	}

	end, err := time.ParseInLocation("2006-01-02", v.EndAt, shanghaiLocation)
	if err != nil {
		return err
	}

	t.StartAt = start
	t.EndAt = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	return nil
}

type GetTermListReq struct {
	Year uint64 `form:"year"`
	Type string `form:"type"`
}

//--------------------------------------------------
