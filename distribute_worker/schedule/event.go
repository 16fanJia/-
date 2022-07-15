package schedule

import "cronTab/common"

const (
	SAVE = iota + 1
	DELETE
	KILL
)

// JobEvent 变化事件
type JobEvent struct {
	EventType int //事件类型 SAVE / DELETE
	Job       common.Job
}

func BuildJobEvent(eventType int, job common.Job) *JobEvent {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}
