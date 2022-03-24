package main

type Record struct {
	TaskID  int64  `json:"task_id"`
	OldReq  []byte `json:"old_req"`
	OldResp []byte `json:"old_resp"`
	NewResp []byte `json:"new_resp"`
}
