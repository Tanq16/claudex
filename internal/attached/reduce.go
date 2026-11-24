package attached

import (
	"bytes"
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
	"time"
)

const (
	StateStarting       = "starting"
	StateIdle           = "idle"
	StateRunning        = "running"
	StateRequiresAction = "requires_action"
	StateExited         = "exited"
)

type Task struct {
	ID           string         `json:"task_id"`
	Type         string         `json:"task_type"`
	Description  string         `json:"description"`
	Ambient      bool           `json:"ambient,omitzero"`
	ToolUseID    string         `json:"tool_use_id,omitzero"`
	WorkflowName string         `json:"workflow_name,omitzero"`
	StartedAt    int64          `json:"startedAt,omitzero"`
	Progress     jsontext.Value `json:"progress,omitzero"`
}

type Todo struct {
	ID         string `json:"id"`
	Subject    string `json:"subject"`
	ActiveForm string `json:"activeForm,omitzero"`
	Status     string `json:"status"`
}

type Context struct {
	TotalTokens  int64          `json:"totalTokens"`
	MaxTokens    int64          `json:"maxTokens"`
	RawMaxTokens int64          `json:"rawMaxTokens,omitzero"`
	Percentage   float64        `json:"percentage"`
	Model        string         `json:"model,omitzero"`
	Categories   jsontext.Value `json:"categories,omitzero"`
}

type Exit struct {
	Code   int    `json:"code"`
	Error  string `json:"error,omitzero"`
	Stderr string `json:"stderr,omitzero"`
}

type Snapshot struct {
	Name           string           `json:"name"`
	Epoch          string           `json:"epoch"`
	PID            int              `json:"pid"`
	StartedAt      int64            `json:"startedAt"`
	State          string           `json:"state"`
	Turn           bool             `json:"turn"`
	Status         string           `json:"status,omitzero"`
	SessionID      string           `json:"sessionId,omitzero"`
	Model          string           `json:"model,omitzero"`
	ResolvedModel  string           `json:"resolvedModel,omitzero"`
	Effort         string           `json:"effort,omitzero"`
	AppliedEffort  string           `json:"appliedEffort,omitzero"`
	PermissionMode string           `json:"permissionMode,omitzero"`
	Context        *Context         `json:"context,omitzero"`
	Limits         jsontext.Value   `json:"limits,omitzero"`
	Tasks          []Task           `json:"tasks"`
	Todos          []Todo           `json:"todos"`
	Pending        []jsontext.Value `json:"pending"`
	Exit           *Exit            `json:"exit,omitzero"`
}

type pendingRequest struct {
	id      string
	subtype string
	tool    string
	raw     jsontext.Value
}

type todoInput struct {
	Subject    string `json:"subject"`
	ActiveForm string `json:"activeForm"`
	TaskID     string `json:"taskId"`
	Status     string `json:"status"`
}

type todoCall struct {
	tool  string
	input todoInput
}

type modelInfo struct {
	Value         string `json:"value"`
	ResolvedModel string `json:"resolvedModel"`
}

type reduced struct {
	snap      Snapshot
	init      jsontext.Value
	models    []modelInfo
	taskInfo  map[string]Task
	todoCalls map[string]todoCall
	pending   []pendingRequest
}

type change struct {
	snapshot bool
	settings bool
	reset    bool
	anchor   string
}

var (
	lenient    = json.JoinOptions(jsontext.AllowInvalidUTF8(true), jsontext.AllowDuplicateNames(true))
	todoMarker = [][]byte{[]byte(`"TaskCreate"`), []byte(`"TaskUpdate"`)}
)

func (r *reduced) reduce(kind string, line []byte) change {
	switch kind {
	case "system/session_state_changed":
		var m struct {
			State string `json:"state"`
		}
		if json.Unmarshal(line, &m, lenient) == nil && m.State != "" {
			r.snap.State = m.State
			if m.State == StateIdle {
				r.snap.Turn = false
			}
			return change{snapshot: true}
		}
	case "system/status":
		var m struct {
			Status         *string `json:"status"`
			PermissionMode string  `json:"permissionMode"`
		}
		if json.Unmarshal(line, &m, lenient) == nil {
			r.snap.Status = ""
			if m.Status != nil {
				r.snap.Status = *m.Status
			}
			return change{snapshot: true, settings: r.setPermissionMode(m.PermissionMode)}
		}
	case "system/init":
		var m struct {
			SessionID      string `json:"session_id"`
			Model          string `json:"model"`
			PermissionMode string `json:"permissionMode"`
		}
		if json.Unmarshal(line, &m, lenient) == nil {
			r.snap.Turn = true
			r.snap.SessionID = cmp.Or(m.SessionID, r.snap.SessionID)
			settings := r.resolveModel(m.Model)
			settings = r.setPermissionMode(m.PermissionMode) || settings
			return change{snapshot: true, settings: settings}
		}
	case "conversation_reset":
		var m struct {
			NewConversationID string `json:"new_conversation_id"`
		}
		if json.Unmarshal(line, &m, lenient) == nil {
			r.snap.SessionID = m.NewConversationID
			r.snap.Todos = []Todo{}
			clear(r.todoCalls)
			return change{snapshot: true, reset: true}
		}
	case "system/background_tasks_changed":
		var m struct {
			Tasks []Task `json:"tasks"`
		}
		if json.Unmarshal(line, &m, lenient) == nil {
			r.snap.Tasks = make([]Task, 0, len(m.Tasks))
			for _, t := range m.Tasks {
				r.snap.Tasks = append(r.snap.Tasks, r.enrich(t))
			}
			return change{snapshot: true}
		}
	case "system/task_started":
		var m struct {
			TaskID       string `json:"task_id"`
			ToolUseID    string `json:"tool_use_id"`
			WorkflowName string `json:"workflow_name"`
		}
		if json.Unmarshal(line, &m, lenient) == nil && m.TaskID != "" {
			info := r.taskInfo[m.TaskID]
			info.ToolUseID, info.WorkflowName, info.StartedAt = m.ToolUseID, m.WorkflowName, time.Now().UnixMilli()
			r.taskInfo[m.TaskID] = info
			return change{snapshot: r.refreshTask(m.TaskID)}
		}
	case "system/task_progress":
		var m struct {
			TaskID string `json:"task_id"`
		}
		if json.Unmarshal(line, &m, lenient) == nil && m.TaskID != "" {
			info := r.taskInfo[m.TaskID]
			info.Progress = jsontext.Value(line)
			r.taskInfo[m.TaskID] = info
			return change{snapshot: r.refreshTask(m.TaskID)}
		}
	case "system/task_notification":
		var m struct {
			TaskID string `json:"task_id"`
		}
		if json.Unmarshal(line, &m, lenient) == nil {
			delete(r.taskInfo, m.TaskID)
		}
	case "result":
		if r.snap.Turn {
			r.snap.Turn = false
			return change{snapshot: true}
		}
	case "rate_limit_event":
		var m struct {
			Info jsontext.Value `json:"rate_limit_info"`
		}
		if json.Unmarshal(line, &m, lenient) == nil && len(m.Info) > 0 {
			r.snap.Limits = m.Info
			return change{snapshot: true}
		}
	case "assistant":
		return r.assistant(line)
	case "user":
		return r.toolResults(line)
	}
	return change{}
}

func (r *reduced) assistant(line []byte) change {
	var m struct {
		UUID            string  `json:"uuid"`
		ParentToolUseID *string `json:"parent_tool_use_id"`
		Message         struct {
			Model string `json:"model"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &m, lenient) != nil {
		return change{}
	}
	var c change
	if m.ParentToolUseID == nil && m.Message.Model != "<synthetic>" {
		c.anchor = m.UUID
	}
	if !hasTodoTool(line) {
		return c
	}
	var blocks struct {
		Message struct {
			Content []struct {
				Type  string         `json:"type"`
				ID    string         `json:"id"`
				Name  string         `json:"name"`
				Input jsontext.Value `json:"input"`
			} `json:"content"`
		} `json:"message"`
	}
	json.Unmarshal(line, &blocks, lenient)
	for _, b := range blocks.Message.Content {
		if b.Type != "tool_use" || (b.Name != "TaskCreate" && b.Name != "TaskUpdate") {
			continue
		}
		var in todoInput
		json.Unmarshal(b.Input, &in, lenient)
		r.todoCalls[b.ID] = todoCall{tool: b.Name, input: in}
	}
	return c
}

func (r *reduced) toolResults(line []byte) change {
	if len(r.todoCalls) == 0 {
		return change{}
	}
	var m struct {
		Message struct {
			Content jsontext.Value `json:"content"`
		} `json:"message"`
		Result jsontext.Value `json:"tool_use_result"`
	}
	if json.Unmarshal(line, &m, lenient) != nil || m.Message.Content.Kind() != '[' {
		return change{}
	}
	var blocks []struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
		IsError   bool   `json:"is_error"`
	}
	if json.Unmarshal(m.Message.Content, &blocks, lenient) != nil {
		return change{}
	}
	var c change
	for _, b := range blocks {
		call, ok := r.todoCalls[b.ToolUseID]
		if b.Type != "tool_result" || !ok {
			continue
		}
		delete(r.todoCalls, b.ToolUseID)
		if !b.IsError && r.applyTodo(call, m.Result) {
			c.snapshot = true
		}
	}
	return c
}

func (r *reduced) applyTodo(call todoCall, result jsontext.Value) bool {
	if call.tool == "TaskCreate" {
		var res struct {
			Task struct {
				ID      string `json:"id"`
				Subject string `json:"subject"`
			} `json:"task"`
		}
		if json.Unmarshal(result, &res, lenient) != nil || res.Task.ID == "" {
			return false
		}
		r.snap.Todos = append(r.snap.Todos, Todo{
			ID:         res.Task.ID,
			Subject:    cmp.Or(res.Task.Subject, call.input.Subject),
			ActiveForm: call.input.ActiveForm,
			Status:     "pending",
		})
		return true
	}
	var res struct {
		Success bool   `json:"success"`
		TaskID  string `json:"taskId"`
	}
	if json.Unmarshal(result, &res, lenient) != nil || !res.Success {
		return false
	}
	id := cmp.Or(res.TaskID, call.input.TaskID)
	i := slices.IndexFunc(r.snap.Todos, func(t Todo) bool { return t.ID == id })
	if i < 0 {
		return false
	}
	if call.input.Status == "deleted" {
		r.snap.Todos = slices.Delete(r.snap.Todos, i, i+1)
		return true
	}
	t := &r.snap.Todos[i]
	t.Status = cmp.Or(call.input.Status, t.Status)
	t.Subject = cmp.Or(call.input.Subject, t.Subject)
	t.ActiveForm = cmp.Or(call.input.ActiveForm, t.ActiveForm)
	return true
}

func (r *reduced) enrich(t Task) Task {
	info, ok := r.taskInfo[t.ID]
	if !ok {
		return t
	}
	t.ToolUseID = cmp.Or(t.ToolUseID, info.ToolUseID)
	t.WorkflowName = cmp.Or(t.WorkflowName, info.WorkflowName)
	t.StartedAt, t.Progress = info.StartedAt, info.Progress
	return t
}

func (r *reduced) refreshTask(id string) bool {
	i := slices.IndexFunc(r.snap.Tasks, func(t Task) bool { return t.ID == id })
	if i < 0 {
		return false
	}
	r.snap.Tasks[i] = r.enrich(r.snap.Tasks[i])
	return true
}

func (r *reduced) tasksRunning() bool {
	return slices.ContainsFunc(r.snap.Tasks, func(t Task) bool { return !t.Ambient })
}

func (r *reduced) setPermissionMode(mode string) bool {
	if mode == "" || mode == r.snap.PermissionMode {
		return false
	}
	r.snap.PermissionMode = mode
	return true
}

func (r *reduced) resolveModel(resolved string) bool {
	if resolved == "" {
		return false
	}
	r.snap.ResolvedModel = resolved
	alias := cmp.Or(r.snap.Model, "default")
	if alias == resolved || len(r.models) == 0 {
		return false
	}
	if slices.Contains(r.models, modelInfo{Value: alias, ResolvedModel: resolved}) {
		return false
	}
	for _, m := range r.models {
		if m.ResolvedModel == resolved && m.Value != "default" {
			r.snap.Model = m.Value
			return true
		}
	}
	return false
}

func (r *reduced) applyInit(reply jsontext.Value) {
	var m struct {
		Models         []modelInfo `json:"models"`
		PermissionMode string      `json:"current_permission_mode"`
	}
	if json.Unmarshal(reply, &m, lenient) != nil {
		return
	}
	r.init, r.models = reply, m.Models
	if r.snap.PermissionMode == "" {
		r.snap.PermissionMode = m.PermissionMode
	}
}

func (r *reduced) addPending(id, subtype, tool string, raw []byte) {
	r.pending = append(r.pending, pendingRequest{id: id, subtype: subtype, tool: tool, raw: jsontext.Value(raw)})
}

func (r *reduced) takePending(id string) (pendingRequest, bool) {
	i := slices.IndexFunc(r.pending, func(p pendingRequest) bool { return p.id == id })
	if i < 0 {
		return pendingRequest{}, false
	}
	p := r.pending[i]
	r.pending = slices.Delete(r.pending, i, i+1)
	return p, true
}

func (r *reduced) public() Snapshot {
	s := r.snap
	s.Tasks = slices.Clone(r.snap.Tasks)
	s.Todos = slices.Clone(r.snap.Todos)
	s.Pending = make([]jsontext.Value, len(r.pending))
	for i, p := range r.pending {
		s.Pending[i] = p.raw
	}
	return s
}

func hasTodoTool(line []byte) bool {
	return slices.ContainsFunc(todoMarker, func(m []byte) bool { return bytes.Contains(line, m) })
}
